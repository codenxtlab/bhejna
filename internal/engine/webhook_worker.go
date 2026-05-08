package engine

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

type ClientWebhookPool struct {
	db     *db.DB
	client *http.Client
	wg     sync.WaitGroup
}

func NewClientWebhookPool(database *db.DB) *ClientWebhookPool {
	return &ClientWebhookPool{
		db: database,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *ClientWebhookPool) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Stop blocks until all webhook workers have finished their current jobs.
func (p *ClientWebhookPool) Stop() {
	p.wg.Wait()
}

func (p *ClientWebhookPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Printf("[WebhookWorker %d] started", id)

	// Outer loop: restarts the inner loop if a panic occurs.
	for {
		select {
		case <-ctx.Done():
			log.Printf("[WebhookWorker %d] stopping", id)
			return
		default:
		}

		exited := p.workerLoop(ctx, id)
		if exited {
			return
		}
		// Panicked and recovered — brief backoff before restarting.
		time.Sleep(500 * time.Millisecond)
	}
}

// workerLoop runs the main processing loop. Returns true if ctx was cancelled,
// false if it exited due to a recovered panic.
func (p *ClientWebhookPool) workerLoop(ctx context.Context, id int) (exited bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: WebhookWorker %d panicked and will restart: %v", id, r)
			exited = false
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WebhookWorker %d] stopping", id)
			return true
		default:
			job, err := p.db.ClaimClientWebhook()
			if err != nil {
				log.Printf("[WebhookWorker %d] Error claiming job: %v", id, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if job == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			p.processJob(id, job)
		}
	}
}

func (p *ClientWebhookPool) processJob(workerID int, job *db.ClientWebhookJob) {
	if job.WebhookURL == "" {
		if err := p.db.MarkClientWebhookSuccess(job.ID); err != nil {
			log.Printf("[WebhookWorker %d] ERROR: failed to mark job %s as success (no URL): %v", workerID, job.ID, err)
		}
		return
	}

	req, err := http.NewRequest("POST", job.WebhookURL, bytes.NewBufferString(job.Payload))
	if err != nil {
		log.Printf("[WebhookWorker %d] Error creating request: %v", workerID, err)
		p.handleFailure(workerID, job)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Bhejna-Webhook-Egress/1.0")

	if job.WebhookSecret != "" {
		sig := calculateHMAC(job.Payload, job.WebhookSecret)
		req.Header.Set("X-Bhejna-Signature", sig)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[WebhookWorker %d] Request failed for job %s: %v", workerID, job.ID, err)
		p.handleFailure(workerID, job)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := p.db.MarkClientWebhookSuccess(job.ID); err != nil {
			log.Printf("[WebhookWorker %d] ERROR: failed to mark job %s as success: %v", workerID, job.ID, err)
		}
	} else {
		log.Printf("[WebhookWorker %d] Job %s failed with status %d", workerID, job.ID, resp.StatusCode)
		p.handleFailure(workerID, job)
	}
}

func (p *ClientWebhookPool) handleFailure(workerID int, job *db.ClientWebhookJob) {
	if job.RetryCount >= 5 {
		log.Printf("[WebhookWorker %d] Job %s exhausted all retries, abandoning", workerID, job.ID)
		return
	}

	backoff := time.Duration(math.Pow(2, float64(job.RetryCount))) * time.Minute
	nextRetry := time.Now().UTC().Add(backoff)
	if err := p.db.MarkClientWebhookFailed(job.ID, job.RetryCount+1, nextRetry); err != nil {
		log.Printf("[WebhookWorker %d] ERROR: failed to mark job %s for retry: %v", workerID, job.ID, err)
	}
}

func calculateHMAC(payload, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
