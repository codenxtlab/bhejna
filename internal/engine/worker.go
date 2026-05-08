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

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: stopping", id)
			return
		default:
			// 1. Claim next job
			job, err := p.db.ClaimNextJob()
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
			tenant, err := p.db.GetTenant(job.TenantID)
			if err != nil {
				log.Printf("Worker %d: error fetching tenant %s: %v", id, job.TenantID, err)
				p.db.RequeueWithJitter(job.ID)
				continue
			}

			// 2. Check rate limit
			if !p.limiters.Allow(tenant.ID, tenant.PhoneNumberID) {
				p.db.RequeueWithJitter(job.ID)
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
					p.db.RequeueWithJitter(job.ID)
				} else if IsPolicyError(err) {
					// 5. Handle policy error
					log.Printf("Worker %d: policy violation for job %s, failing job and pausing tenant %s", id, job.ID, job.TenantID)
					p.db.MarkJobFailed(job.ID, errorCode, errorMessage)
					p.db.PauseTenant(job.TenantID, "POLICY_VIOLATION")
				} else {
					log.Printf("Worker %d: permanent failure for job %s: %v", id, job.ID, err)
					p.db.MarkJobFailed(job.ID, errorCode, errorMessage)
				}
				continue
			}

			// 6. On success: Update job with wamid and set status to accepted (Level 2)
			// We first bind the wamid to the internal job ID
			_ = p.db.SetJobMetaID(job.ID, wamid)
			p.db.UpdateJobMonotonic(wamid, "accepted", 2)
		}
	}
}
