package domain

import (
	"context"
)

// ImageGenerationRequest represents the parameters for image generation
type ImageGenerationRequest struct {
	Prompt         string
	Width          int
	Height         int
	NumImages      int
	Style          string
	NegativePrompt string
}

// ImageGenerationResponse represents the response from the image generation service
type ImageGenerationResponse struct {
	UUID             string
	Status           string
	Files            []string
	Censored         bool
	ErrorDescription string
}

// ImageGenerationService defines the interface for image generation operations
type ImageGenerationService interface {
	// GenerateImage generates an image based on the provided prompt
	GenerateImage(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error)

	// CheckGenerationStatus checks the status of an image generation request
	CheckGenerationStatus(ctx context.Context, uuid string) (*ImageGenerationResponse, error)
}
