package service

import (
	"context"
	"fmt"
	"time"

	"github.com/basel-ax/2xiang/internal/config"
	"github.com/basel-ax/2xiang/internal/domain"
	"github.com/basel-ax/2xiang/internal/infrastructure/fusionbrain"
)

// ImageGenerationService implements the domain.ImageGenerationService interface
type ImageGenerationService struct {
	client *fusionbrain.Client
	config *config.Config
}

// NewImageGenerationService creates a new image generation service
func NewImageGenerationService(cfg *config.Config) *ImageGenerationService {
	return &ImageGenerationService{
		client: fusionbrain.NewClient(cfg.FusionBrainAPIKey, cfg.FusionBrainSecretKey),
		config: cfg,
	}
}

// GenerateImage implements the image generation request
func (s *ImageGenerationService) GenerateImage(ctx context.Context, req domain.ImageGenerationRequest) (*domain.ImageGenerationResponse, error) {
	// Set default values if not provided
	if req.Width == 0 {
		req.Width = s.config.DefaultImageWidth
	}
	if req.Height == 0 {
		req.Height = s.config.DefaultImageHeight
	}
	if req.NumImages == 0 {
		req.NumImages = s.config.DefaultNumImages
	}

	// Generate the image
	resp, err := s.client.GenerateImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return resp, nil
}

// CheckGenerationStatus checks the status of an image generation request
func (s *ImageGenerationService) CheckGenerationStatus(ctx context.Context, uuid string) (*domain.ImageGenerationResponse, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, s.config.CheckInterval)
	defer cancel()

	resp, err := s.client.CheckGenerationStatus(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to check generation status: %w", err)
	}

	return resp, nil
}

// WaitForGeneration waits for the image generation to complete
func (s *ImageGenerationService) WaitForGeneration(ctx context.Context, uuid string) (*domain.ImageGenerationResponse, error) {
	for i := 0; i < s.config.MaxAttempts; i++ {
		resp, err := s.CheckGenerationStatus(ctx, uuid)
		if err != nil {
			return nil, fmt.Errorf("failed to check generation status: %w", err)
		}

		switch resp.Status {
		case "DONE":
			return resp, nil
		case "FAIL":
			return nil, fmt.Errorf("generation failed: %s", resp.ErrorDescription)
		case "INITIAL", "PROCESSING":
			// Wait before next attempt
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.config.CheckInterval):
				continue
			}
		default:
			return nil, fmt.Errorf("unknown status: %s", resp.Status)
		}
	}

	return nil, fmt.Errorf("max attempts reached waiting for generation")
}
