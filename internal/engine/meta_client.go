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

// MetaResponse represents the success response from WhatsApp Cloud API.
type MetaResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Messages         []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// MetaErrorResponse represents an error response from WhatsApp Cloud API.
type MetaErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		ErrorData struct {
			Details string `json:"details"`
		} `json:"error_data"`
		FBTraceID string `json:"fbtrace_id"`
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
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", phoneNumberID)

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

	var metaResp MetaResponse
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
