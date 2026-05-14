# Graph Report - /home/rakshitbhai/bhejna  (2026-05-14)

## Corpus Check
- 20 files · ~16,403 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 237 nodes · 366 edges · 25 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 97 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]

## God Nodes (most connected - your core abstractions)
1. `DB` - 29 edges
2. `main()` - 22 edges
3. `Handler()` - 14 edges
4. `SyncTenantJSONBody` - 9 edges
5. `HandleSendMessage()` - 8 edges
6. `ClientWebhookPool` - 7 edges
7. `HandleWebhookEvent()` - 7 edges
8. `HandleSyncTenant()` - 7 edges
9. `Server` - 7 edges
10. `ServerInterfaceWrapper` - 7 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewLimiterManager()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/limiter.go
- `main()` --calls--> `NewMetaAPIClient()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/meta_client.go
- `main()` --calls--> `NewWorkerPool()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/worker.go
- `main()` --calls--> `NewClientWebhookPool()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/engine/webhook_worker.go
- `main()` --calls--> `NewMetaWebhook()`  [INFERRED]
  /home/rakshitbhai/bhejna/cmd/bhejna/main.go → /home/rakshitbhai/bhejna/internal/api/handlers/meta_webhook.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (32): apiKeyAuthContextKey, ErrorResponse, ForceGenerateWebhookTypeRequestObject, ForceGenerateWebhookTypeResponseObject, GetV1MetaWebhookParams, GetV1MetaWebhookRequestObject, GetV1MetaWebhookResponseObject, internalAuthContextKey (+24 more)

### Community 1 - "Community 1"
Cohesion: 0.12
Nodes (6): DB, parkAndSweep(), parseWebhookDummy(), staleDetector(), StartJanitor(), HandleWebhookEvent()

### Community 2 - "Community 2"
Cohesion: 0.1
Nodes (13): contextKey, NewStrictHandler(), HandleSendMessage(), InvalidParamFormatError, TooManyValuesForParamError, UnmarshalingParamError, WebhookPayloadEntryChangesValueStatusesStatus, APIKeyMiddleware() (+5 more)

### Community 3 - "Community 3"
Cohesion: 0.13
Nodes (13): performCleanup(), StartCleanupJanitor(), applyMigrations(), InitDB(), MetaAPIClient, SupabaseJob, getEnv(), main() (+5 more)

### Community 4 - "Community 4"
Cohesion: 0.15
Nodes (7): Handler(), ForceGenerateWebhookType200JSONResponse, PauseTenant204Response, SendMessage429Response, ServerInterfaceWrapper, strictHandler, SyncTenant200Response

### Community 5 - "Community 5"
Cohesion: 0.12
Nodes (4): Server, SyncTenantJSONBody, HandlePauseTenant(), HandleSyncTenant()

### Community 6 - "Community 6"
Cohesion: 0.14
Nodes (12): MetaAPIError, MetaErrorResponse, MetaMessageResponse, SendMessagePayload, TemplateComponent, TemplateContent, TemplateLanguage, TemplateParameter (+4 more)

### Community 7 - "Community 7"
Cohesion: 0.15
Nodes (6): GetV1MetaWebhook200TextResponse, GetV1MetaWebhook403Response, PostV1MetaWebhook200Response, MetaWebhook, extractPhoneNumberID(), NewMetaWebhook()

### Community 8 - "Community 8"
Cohesion: 0.27
Nodes (3): ClientWebhookPool, calculateHMAC(), NewClientWebhookPool()

### Community 9 - "Community 9"
Cohesion: 0.33
Nodes (5): ActiveSession, ClientWebhookJob, Job, Tenant, WebhookEvent

### Community 10 - "Community 10"
Cohesion: 0.4
Nodes (2): WorkerPool, NewWorkerPool()

### Community 11 - "Community 11"
Cohesion: 0.5
Nodes (2): LimiterManager, NewLimiterManager()

### Community 12 - "Community 12"
Cohesion: 0.67
Nodes (1): UnescapedCookieParamError

### Community 13 - "Community 13"
Cohesion: 0.67
Nodes (3): HandlerFromMux(), HandlerFromMuxWithBaseURL(), HandlerWithOptions()

### Community 14 - "Community 14"
Cohesion: 0.67
Nodes (1): RequiredHeaderError

### Community 15 - "Community 15"
Cohesion: 0.67
Nodes (3): GetSpec(), GetSwagger(), PathToRawSpec()

### Community 16 - "Community 16"
Cohesion: 1.0
Nodes (2): decodeSpec(), decodeSpecCached()

### Community 17 - "Community 17"
Cohesion: 1.0
Nodes (1): SendMessageResponseStatus

### Community 18 - "Community 18"
Cohesion: 1.0
Nodes (1): RequiredParamError

### Community 19 - "Community 19"
Cohesion: 1.0
Nodes (1): SendMessage400JSONResponse

### Community 20 - "Community 20"
Cohesion: 1.0
Nodes (1): SendMessage401Response

### Community 21 - "Community 21"
Cohesion: 1.0
Nodes (1): SendMessage202JSONResponse

### Community 22 - "Community 22"
Cohesion: 1.0
Nodes (1): MessageType

### Community 23 - "Community 23"
Cohesion: 1.0
Nodes (0): 

### Community 24 - "Community 24"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **47 isolated node(s):** `Tenant`, `Job`, `WebhookEvent`, `ActiveSession`, `ClientWebhookJob` (+42 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 16`** (2 nodes): `decodeSpec()`, `decodeSpecCached()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 17`** (2 nodes): `SendMessageResponseStatus`, `.Valid()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 18`** (2 nodes): `RequiredParamError`, `.Error()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 19`** (2 nodes): `SendMessage400JSONResponse`, `.VisitSendMessageResponse()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 20`** (2 nodes): `SendMessage401Response`, `.VisitSendMessageResponse()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 21`** (2 nodes): `SendMessage202JSONResponse`, `.VisitSendMessageResponse()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 22`** (2 nodes): `MessageType`, `.Valid()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 23`** (1 nodes): `repository.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 24`** (1 nodes): `generate.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main()` connect `Community 3` to `Community 1`, `Community 2`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 10`, `Community 11`?**
  _High betweenness centrality (0.194) - this node is a cross-community bridge._
- **Why does `decodeSpec()` connect `Community 16` to `Community 0`, `Community 3`?**
  _High betweenness centrality (0.172) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 1` to `Community 8`, `Community 2`, `Community 3`, `Community 5`?**
  _High betweenness centrality (0.135) - this node is a cross-community bridge._
- **Are the 20 inferred relationships involving `main()` (e.g. with `InitDB()` and `.Close()`) actually correct?**
  _`main()` has 20 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `HandleSendMessage()` (e.g. with `GetTenant()` and `.Error()`) actually correct?**
  _`HandleSendMessage()` has 7 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Tenant`, `Job`, `WebhookEvent` to the rest of the system?**
  _47 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._