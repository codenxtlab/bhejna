package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

type WorkerPool struct {
	db          *db.DB
	limiters    *LimiterManager
	metaClient  *MetaAPIClient
	workerCount int
	wg          sync.WaitGroup
}

func NewWorkerPool(database *db.DB, limiters *LimiterManager, metaClient *MetaAPIClient, count int) *WorkerPool {
	return &WorkerPool{
		db:          database,
		limiters:    limiters,
		metaClient:  metaClient,
		workerCount: count,
	}
}

func (p *WorkerPool) Stop() {
	p.wg.Wait()
}

func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Printf("Worker %d: started", id)

	// Outer loop: restarts the inner loop if a panic occurs.
	// The worker only exits permanently when ctx is cancelled.
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: stopping", id)
			return
		default:
		}

		exited := p.workerLoop(ctx, id)
		if exited {
			// Context was cancelled inside the loop, exit cleanly.
			return
		}
		// If we reach here, the inner loop panicked and recovered.
		// Brief backoff before restarting to avoid tight panic loops.
		time.Sleep(500 * time.Millisecond)
	}
}

// workerLoop runs the main processing loop. Returns true if ctx was cancelled,
// false if it exited due to a recovered panic.
func (p *WorkerPool) workerLoop(ctx context.Context, id int) (exited bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: Worker %d panicked and will restart: %v", id, r)
			exited = false
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: stopping", id)
			return true
		default:
			// 1. Claim next job
			job, err := p.db.ClaimNextJob(ctx)
			if err != nil {
				log.Printf("Worker %d: error claiming job: %v", id, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if job == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Fetch tenant for access token and phone ID
			tenant, err := p.db.GetTenant(ctx, job.TenantID)
			if err != nil {
				log.Printf("Worker %d: error fetching tenant %s: %v", id, job.TenantID, err)
				p.db.RequeueWithJitter(ctx, job.ID)
				continue
			}
			if tenant == nil {
				log.Printf("Worker %d: tenant %s not found, failing job %s", id, job.TenantID, job.ID)
				p.db.MarkJobFailed(ctx, job.ID, "TENANT_NOT_FOUND", "tenant was deleted")
				continue
			}

			// 2. Check rate limit
			if !p.limiters.Allow(tenant.ID, tenant.PhoneNumberID) {
				p.db.RequeueWithJitter(ctx, job.ID)
				continue
			}

			// 3. Send message
			wamid, err := p.metaClient.SendMessage(job, tenant.AccessToken, tenant.PhoneNumberID)
			if err != nil {
				errorCode := "UNKNOWN"
				errorMessage := err.Error()

				if apiErr, ok := err.(*MetaAPIError); ok {
					errorCode = fmt.Sprintf("%d", apiErr.Code)
					errorMessage = apiErr.Message
					if apiErr.RawBody != "" {
						errorMessage = apiErr.RawBody
					}
				}

				// 4. Handle transient error
				if IsTransientError(err) {
					log.Printf("Worker %d: transient error for job %s, requeueing: %v", id, job.ID, err)
					p.db.RequeueWithJitter(ctx, job.ID)
				} else if IsPolicyError(err) {
					// 5. Handle policy error
					log.Printf("Worker %d: policy violation for job %s, failing job and pausing tenant %s", id, job.ID, job.TenantID)
					p.db.MarkJobFailed(ctx, job.ID, errorCode, errorMessage)
					p.db.PauseTenant(ctx, job.TenantID, "POLICY_VIOLATION")
				} else {
					log.Printf("Worker %d: permanent failure for job %s: %v", id, job.ID, err)
					p.db.MarkJobFailed(ctx, job.ID, errorCode, errorMessage)
				}
				continue
			}

			// 6. On success: Bind the wamid to the internal job ID.
			// This is CRITICAL — if SetJobMetaID fails, delivery status webhooks
			// from Meta will never match this job, creating an orphaned record.
			if err := p.db.SetJobMetaID(ctx, job.ID, wamid); err != nil {
				log.Printf("Worker %d: CRITICAL: failed to bind wamid %s to job %s: %v", id, wamid, job.ID, err)
				// Fallback: mark accepted by job ID directly so it doesn't stay stuck in "processing"
				if err := p.db.UpdateJobStatus(ctx, job.ID, "accepted", 2); err != nil {
					log.Printf("Worker %d: ERROR: fallback UpdateJobStatus also failed for job %s: %v", id, job.ID, err)
				}
				continue
			}

			// Monotonic update to "accepted" (Level 2) by wamid
			if _, err := p.db.UpdateJobMonotonic(ctx, wamid, "accepted", 2); err != nil {
				log.Printf("Worker %d: error updating job %s to accepted: %v", id, job.ID, err)
			}
		}
	}
}
