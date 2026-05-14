package api

import (
	"encoding/json"
	"net/http"

	"github.com/codenxtlab/bhejna/internal/api/generated"
	"github.com/codenxtlab/bhejna/internal/api/handlers"
	"github.com/codenxtlab/bhejna/internal/db"
)

// Server implements generated.ServerInterface
type Server struct {
	DB             *db.DB
	InternalSecret string
	WebhookHandler *handlers.MetaWebhook
}

func (s *Server) SyncTenant(w http.ResponseWriter, r *http.Request) {
	HandleSyncTenant(s.DB, s.InternalSecret).ServeHTTP(w, r)
}

func (s *Server) PauseTenant(w http.ResponseWriter, r *http.Request, id string) {
	// The generated interface passes `id` parameter, but our current handler
	// relies on chi.URLParam(r, "id"). So we must ensure it's set in context,
	// or we pass it directly. Since chi is still routing, chi.URLParam will work.
	HandlePauseTenant(s.DB).ServeHTTP(w, r)
}

func (s *Server) ForceGenerateWebhookType(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) SendMessage(w http.ResponseWriter, r *http.Request, params generated.SendMessageParams) {
	// The params are handled by HandleSendMessage internally (reads headers).
	HandleSendMessage(s.DB).ServeHTTP(w, r)
}

func (s *Server) GetV1MetaWebhook(w http.ResponseWriter, r *http.Request, params generated.GetV1MetaWebhookParams) {
	req := generated.GetV1MetaWebhookRequestObject{Params: params}
	res, _ := s.WebhookHandler.GetV1MetaWebhook(r.Context(), req)
	if res != nil {
		res.VisitGetV1MetaWebhookResponse(w)
	}
}

func (s *Server) PostV1MetaWebhook(w http.ResponseWriter, r *http.Request) {
	var payload generated.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	req := generated.PostV1MetaWebhookRequestObject{Body: &payload}
	res, _ := s.WebhookHandler.PostV1MetaWebhook(r.Context(), req)
	if res != nil {
		res.VisitPostV1MetaWebhookResponse(w)
	}
}
