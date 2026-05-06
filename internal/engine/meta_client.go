package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/codenxtlab/bhejna/internal/db"
)

type MetaAPIClient struct {
	client *http.Client
}

func NewMetaAPIClient() *MetaAPIClient {
	return &MetaAPIClient{
		client: &http.Client{},
	}
}

// --- REQUEST STRUCTS ---

type SendMessagePayload struct {
	MessagingProduct string           `json:"messaging_product"` // Always "whatsapp"
	RecipientType    string           `json:"recipient_type"`    // Always "individual"
	To               string           `json:"to"`
	Type             string           `json:"type"`              // "text" or "template"
	Text             *TextContent     `json:"text,omitempty"`
	Template         *TemplateContent `json:"template,omitempty"`
}

type TextContent struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

type TemplateContent struct {
	Name       string             `json:"name"`
	Language   TemplateLanguage   `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

type TemplateLanguage struct {
	Code string `json:"code"` // e.g., "en_US"
}

type TemplateComponent struct {
	Type       string             `json:"type"` // "header", "body", "button"
	Parameters []TemplateParameter `json:"parameters"`
}

type TemplateParameter struct {
	Type string `json:"type"` // "text", "currency", "image", etc.
	Text string `json:"text,omitempty"`
}

// --- RESPONSE STRUCTS ---

// MetaMessageResponse is what Meta returns on a 200 OK.
type MetaMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"` // This is the wamid.HBg... you need to save
	} `json:"messages"`
}

// MetaErrorResponse is what Meta returns on 4xx/5xx errors.
type MetaErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		ErrorData struct {
			Details string `json:"details"`
		} `json:"error_data"`
		ErrorSubcode int `json:"error_subcode"` // Critical for identifying 131048 limits
	} `json:"error"`
}

type MetaAPIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *MetaAPIError) Error() string {
	return fmt.Sprintf("meta api error (status %d, code %d): %s", e.StatusCode, e.Code, e.Message)
}

func (c *MetaAPIClient) SendMessage(job *db.Job, accessToken string, phoneNumberID string) (string, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v25.0/%s/messages", phoneNumberID)

	req, err := http.NewRequest("POST", url, strings.NewReader(job.MessagePayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errResp MetaErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return "", fmt.Errorf("meta api error (status %d): %s", resp.StatusCode, string(body))
		}
		return "", &MetaAPIError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Error.Code,
			Message:    errResp.Error.Message,
		}
	}

	var metaResp MetaMessageResponse
	if err := json.Unmarshal(body, &metaResp); err != nil {
		return "", fmt.Errorf("failed to decode meta response: %v", err)
	}

	if len(metaResp.Messages) == 0 {
		return "", fmt.Errorf("no message id returned in meta response")
	}

	return metaResp.Messages[0].ID, nil
}

// IsTransientError returns true for 5xx errors or network failures.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*MetaAPIError); ok {
		return apiErr.StatusCode >= 500
	}
	return true
}

// IsPolicyError returns true for 4xx errors or specific policy violation codes.
func IsPolicyError(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*MetaAPIError); ok {
		// 131048: Rate limit hit
		return apiErr.StatusCode == 400 || apiErr.Code == 131048
	}
	return false
}
