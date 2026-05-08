# Graph Report - /home/rakshitbhai/bhejna  (2026-05-08)

## Corpus Check
- 15 files · ~5,496 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 102 nodes · 169 edges · 15 communities detected
- Extraction: 59% EXTRACTED · 41% INFERRED · 0% AMBIGUOUS · INFERRED: 69 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]

## God Nodes (most connected - your core abstractions)
1. `DB` - 26 edges
2. `main()` - 21 edges
3. `HandleWebhookEvent()` - 8 edges
4. `parkAndSweep()` - 6 edges
5. `StartJanitor()` - 5 edges
6. `syncJobs()` - 5 edges
7. `ClientWebhookPool` - 5 edges
8. `HandleSendMessage()` - 5 edges
9. `WorkerPool` - 4 edges
10. `StartSupabaseSync()` - 4 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewLimiterManager()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/limiter.go
- `main()` --calls--> `NewMetaAPIClient()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/meta_client.go
- `main()` --calls--> `NewWorkerPool()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/worker.go
- `main()` --calls--> `NewClientWebhookPool()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/webhook_worker.go
- `main()` --calls--> `InitDB()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/db/db.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.22
Nodes (2): DB, HandleWebhookEvent()

### Community 1 - "Community 1"
Cohesion: 0.14
Nodes (12): MetaAPIClient, MetaErrorResponse, MetaMessageResponse, SendMessagePayload, TemplateComponent, TemplateContent, TemplateLanguage, TemplateParameter (+4 more)

### Community 2 - "Community 2"
Cohesion: 0.29
Nodes (3): ClientWebhookPool, calculateHMAC(), NewClientWebhookPool()

### Community 3 - "Community 3"
Cohesion: 0.31
Nodes (3): SupabaseJob, StartSupabaseSync(), syncJobs()

### Community 4 - "Community 4"
Cohesion: 0.25
Nodes (4): performCleanup(), StartCleanupJanitor(), WorkerPool, NewWorkerPool()

### Community 5 - "Community 5"
Cohesion: 0.33
Nodes (5): ActiveSession, ClientWebhookJob, Job, Tenant, WebhookEvent

### Community 6 - "Community 6"
Cohesion: 0.4
Nodes (4): contextKey, APIKeyMiddleware(), MetaSignatureMiddleware(), validateSignature()

### Community 7 - "Community 7"
Cohesion: 0.4
Nodes (4): HandleSendMessage(), MetaAPIError, GetTenant(), InternalJWTMiddleware()

### Community 8 - "Community 8"
Cohesion: 0.53
Nodes (4): parkAndSweep(), parseWebhookDummy(), staleDetector(), StartJanitor()

### Community 9 - "Community 9"
Cohesion: 0.5
Nodes (3): InitDB(), getEnv(), main()

### Community 10 - "Community 10"
Cohesion: 0.5
Nodes (2): HandlePauseTenant(), HandleSyncTenant()

### Community 11 - "Community 11"
Cohesion: 0.5
Nodes (2): LimiterManager, NewLimiterManager()

### Community 12 - "Community 12"
Cohesion: 0.67
Nodes (2): SendMessageRequest, SendMessageResponse

### Community 13 - "Community 13"
Cohesion: 0.67
Nodes (2): WebhookPayload, HandleWebhookValidation()

### Community 14 - "Community 14"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **18 isolated node(s):** `Tenant`, `Job`, `WebhookEvent`, `ActiveSession`, `ClientWebhookJob` (+13 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 14`** (1 nodes): `repository.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `Community 9` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 6`, `Community 7`, `Community 8`, `Community 10`, `Community 11`, `Community 13`?**
  _High betweenness centrality (0.382) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 0` to `Community 2`, `Community 3`, `Community 4`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`?**
  _High betweenness centrality (0.232) - this node is a cross-community bridge._
- **Why does `NewMetaAPIClient()` connect `Community 1` to `Community 9`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Are the 19 inferred relationships involving `main()` (e.g. with `InitDB()` and `.Close()`) actually correct?**
  _`main()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `HandleWebhookEvent()` (e.g. with `.UpdateJobMonotonic()` and `.GetTenantByWabaID()`) actually correct?**
  _`HandleWebhookEvent()` has 7 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `parkAndSweep()` (e.g. with `.GetUnmatchedEvents()` and `.UpdateJobMonotonic()`) actually correct?**
  _`parkAndSweep()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `StartJanitor()` (e.g. with `.Stop()` and `main()`) actually correct?**
  _`StartJanitor()` has 2 INFERRED edges - model-reasoned connections that need verification._