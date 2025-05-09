package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/swenro11/2xiang/internal/config"
	"github.com/swenro11/2xiang/internal/domain"
	"github.com/swenro11/2xiang/internal/repository"
	"github.com/swenro11/2xiang/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)

	// Create repository and service
	imgRepo := repository.NewPostgresImageRepository(db)
	imgService := service.NewImageGenerationService(cfg)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GenerationTimeout)
	defer cancel()

	// Start the workflows
	go generateImagesWorkflow(ctx, imgRepo, imgService, cfg)
	go processGeneratedImagesWorkflow(ctx, imgRepo, imgService)

	// Keep the main goroutine running
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	}
}

// generateImagesWorkflow handles the generation of new images
func generateImagesWorkflow(ctx context.Context, repo repository.ImageRepository, service *service.ImageGenerationService, cfg *config.Config) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			img, err := repo.GetReadyToGenerate(ctx)
			if err != nil {
				log.Printf("Error getting ready image: %v", err)
				time.Sleep(time.Second)
				continue
			}
			if img == nil {
				time.Sleep(time.Second)
				continue
			}

			// Create image generation request
			req := domain.ImageGenerationRequest{
				Prompt:         img.Prompt,
				Width:          cfg.DefaultImageWidth,
				Height:         cfg.DefaultImageHeight,
				NumImages:      cfg.DefaultNumImages,
				Style:          cfg.DefaultStyle,
				NegativePrompt: cfg.DefaultNegativePrompt,
			}

			// Generate image
			resp, err := service.GenerateImage(ctx, req)
			if err != nil {
				log.Printf("Error generating image: %v", err)
				if err := repo.UpdateStatus(ctx, img.ID, "Failed"); err != nil {
					log.Printf("Error updating status: %v", err)
				}
				continue
			}

			// Update image status and UUID
			if err := repo.UpdateStatus(ctx, img.ID, "Generate"); err != nil {
				log.Printf("Error updating status: %v", err)
				continue
			}
			if err := repo.UpdateUUID(ctx, img.ID, resp.UUID); err != nil {
				log.Printf("Error updating UUID: %v", err)
				continue
			}
		}
	}
}

// processGeneratedImagesWorkflow handles checking and processing generated images
func processGeneratedImagesWorkflow(ctx context.Context, repo repository.ImageRepository, service *service.ImageGenerationService) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get an image ready for status check
			img, err := repo.GetReadyToCheck(ctx)
			if err != nil {
				log.Printf("Error getting image ready for check: %v", err)
				continue
			}
			if img == nil {
				continue
			}

			// Check generation status
			resp, err := service.CheckGenerationStatus(ctx, img.UUID)
			if err != nil {
				log.Printf("Error checking generation status: %v", err)
				continue
			}

			// If generation is complete, save the base64 data
			if resp.Status == "DONE" && len(resp.Files) > 0 {
				if err := repo.UpdateBase64(ctx, img.ID, resp.Files[0]); err != nil {
					log.Printf("Error updating image base64: %v", err)
					continue
				}
				if err := repo.UpdateStatus(ctx, img.ID, "Completed"); err != nil {
					log.Printf("Error updating image status: %v", err)
					continue
				}
			} else if resp.Status == "FAILED" {
				if err := repo.UpdateStatus(ctx, img.ID, "Failed"); err != nil {
					log.Printf("Error updating image status: %v", err)
					continue
				}
			}
		}
	}
}
