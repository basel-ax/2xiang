package fusionbrain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/swenro11/2xiang/internal/domain"
)

const (
	baseURL = "https://api-key.fusionbrain.ai"
)

// Client represents the Fusion Brain API client
type Client struct {
	httpClient *http.Client
	apiKey     string
	secretKey  string
}

// NewClient creates a new Fusion Brain API client
func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:    apiKey,
		secretKey: secretKey,
	}
}

// GenerateImage implements the image generation request
func (c *Client) GenerateImage(ctx context.Context, req domain.ImageGenerationRequest) (*domain.ImageGenerationResponse, error) {
	// First, get the pipeline ID
	pipelineID, err := c.getPipelineID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline ID: %w", err)
	}

	// Prepare the request body
	params := map[string]interface{}{
		"type":      "GENERATE",
		"width":     req.Width,
		"height":    req.Height,
		"numImages": req.NumImages,
		"generateParams": map[string]string{
			"query": req.Prompt,
		},
	}

	if req.Style != "" {
		params["style"] = req.Style
	}

	if req.NegativePrompt != "" {
		params["negativePromptDecoder"] = req.NegativePrompt
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add pipeline_id
	if err := writer.WriteField("pipeline_id", pipelineID); err != nil {
		return nil, fmt.Errorf("failed to write pipeline_id: %w", err)
	}

	// Add params as JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := writer.WriteField("params", string(paramsJSON)); err != nil {
		return nil, fmt.Errorf("failed to write params: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/key/api/v1/pipeline/run", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-Key", "Key "+c.apiKey)
	httpReq.Header.Set("X-Secret", "Secret "+c.secretKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		UUID   string `json:"uuid"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &domain.ImageGenerationResponse{
		UUID:   result.UUID,
		Status: result.Status,
	}, nil
}

// CheckGenerationStatus checks the status of an image generation request
func (c *Client) CheckGenerationStatus(ctx context.Context, uuid string) (*domain.ImageGenerationResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/key/api/v1/pipeline/status/%s", baseURL, uuid), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-Key", "Key "+c.apiKey)
	httpReq.Header.Set("X-Secret", "Secret "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		UUID             string `json:"uuid"`
		Status           string `json:"status"`
		ErrorDescription string `json:"errorDescription"`
		Result           struct {
			Files    []string `json:"files"`
			Censored bool     `json:"censored"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &domain.ImageGenerationResponse{
		UUID:             result.UUID,
		Status:           result.Status,
		Files:            result.Result.Files,
		Censored:         result.Result.Censored,
		ErrorDescription: result.ErrorDescription,
	}, nil
}

// getPipelineID retrieves the pipeline ID for the Kandinsky model
func (c *Client) getPipelineID(ctx context.Context) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/key/api/v1/pipelines", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-Key", "Key "+c.apiKey)
	httpReq.Header.Set("X-Secret", "Secret "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var pipelines []struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pipelines); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(pipelines) == 0 {
		return "", fmt.Errorf("no pipelines found")
	}

	return pipelines[0].ID, nil
}
