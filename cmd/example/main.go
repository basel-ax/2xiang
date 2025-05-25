package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/basel-ax/2xiang/internal/config"
	"github.com/basel-ax/2xiang/internal/domain"
	"github.com/basel-ax/2xiang/internal/repository"
	"github.com/basel-ax/2xiang/internal/service"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

const (
	maxPromptLength = 999
)

// truncatePrompt safely truncates a string to the specified length while preserving UTF-8 characters
func truncatePrompt(s string, length int) string {
	if utf8.RuneCountInString(s) <= length {
		return s
	}

	var size, n int
	for i := 0; i < length && n < len(s); i++ {
		_, size = utf8.DecodeRuneInString(s[n:])
		n += size
	}

	return s[:n]
}

func main() {
	// Parse command line flags
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	runGenerator := flag.Bool("generator", false, "Run image generation workflow")
	runProcessor := flag.Bool("processor", false, "Run image processing workflow")
	runCron := flag.Bool("cron", false, "Run workflows on schedule (generator every 5min, processor every 10min)")
	flag.Parse()

	// Configure logging
	if *verbose {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.Println("Verbose logging enabled")
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}

	// Check if at least one workflow is selected
	if !*runGenerator && !*runProcessor && !*runCron {
		log.Fatal("Please specify at least one workflow to run: -generator, -processor, or -cron")
	}

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Println("Configuration loaded successfully")

	// Initialize database connection
	log.Println("Initializing database connection...")
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	log.Println("Database connection established")

	// Initialize repository and service
	imgRepo := repository.NewPostgresImageRepository(db)
	log.Println("Initializing image generation service...")
	imgService := service.NewImageGenerationService(cfg)
	log.Println("Image generation service initialized")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, initiating shutdown...", sig)
		cancel()
	}()

	// Start selected workflows
	if *runCron {
		log.Println("Starting scheduled workflows...")
		startCronWorkflows(ctx, imgRepo, imgService, cfg)
	} else {
		if *runGenerator {
			log.Println("Starting image generation workflow...")
			go generateImagesWorkflow(ctx, imgRepo, imgService, cfg)
		}

		if *runProcessor {
			log.Println("Starting image processing workflow...")
			go processGeneratedImagesWorkflow(ctx, imgRepo, imgService, cfg)
		}
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down gracefully...")
}

func startCronWorkflows(ctx context.Context, repo repository.ImageRepository, service *service.ImageGenerationService, cfg *config.Config) {
	// Create a new cron scheduler
	c := cron.New(cron.WithSeconds())

	var cronMutex sync.Mutex

	// Add generator workflow to run every 5 minutes
	_, err := c.AddFunc("0 */3 * * * *", func() {
		log.Println("[CRON] Attempting to start scheduled generator workflow...")
		cronMutex.Lock()
		defer cronMutex.Unlock()
		log.Println("[CRON] Running scheduled generator workflow...")
		generateImagesWorkflow(ctx, repo, service, cfg)
		log.Println("[CRON] Finished scheduled generator workflow.")
	})
	if err != nil {
		log.Printf("Error scheduling generator workflow: %v", err)
		return
	}

	// Add processor workflow to run every 10 minutes
	_, err = c.AddFunc("0 */7 * * * *", func() {
		log.Println("[CRON] Attempting to start scheduled processor workflow...")
		cronMutex.Lock()
		defer cronMutex.Unlock()
		log.Println("[CRON] Running scheduled processor workflow...")
		processGeneratedImagesWorkflow(ctx, repo, service, cfg)
		log.Println("[CRON] Finished scheduled processor workflow.")
	})
	if err != nil {
		log.Printf("Error scheduling processor workflow: %v", err)
		return
	}

	// Start the cron scheduler
	c.Start()
	log.Println("Cron scheduler started successfully")

	// Keep the scheduler running until context is cancelled
	<-ctx.Done()
	c.Stop()
	log.Println("Cron scheduler stopped")
}

func generateImagesWorkflow(ctx context.Context, repo repository.ImageRepository, service *service.ImageGenerationService, cfg *config.Config) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Image generation workflow stopped")
			return
		case <-ticker.C:
			// Get all images ready for generation
			images, err := repo.GetAllReadyToGenerate(ctx)
			if err != nil {
				log.Printf("Error getting ready images: %v", err)
				continue
			}

			if len(images) == 0 {
				continue
			}

			for _, img := range images {
				// Truncate prompt if it exceeds the maximum length
				originalPrompt := img.Prompt
				img.Prompt = truncatePrompt(img.Prompt, maxPromptLength)
				if len(originalPrompt) != len(img.Prompt) {
					log.Printf("Prompt for image ID %d was truncated from %d to %d characters", img.ID, len(originalPrompt), len(img.Prompt))
				}

				log.Printf("Processing image ID %d with prompt: %s", img.ID, img.Prompt)

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
					// Check if this is an INITIAL status response
					if strings.Contains(err.Error(), "status code: 201") && strings.Contains(err.Error(), "INITIAL") {
						// Extract UUID from error message
						uuidMatch := regexp.MustCompile(`"uuid":"([^"]+)"`).FindStringSubmatch(err.Error())
						if len(uuidMatch) > 1 {
							uuid := uuidMatch[1]
							log.Printf("Image generation initiated for ID %d with UUID: %s", img.ID, uuid)

							// Update image UUID
							if err := repo.UpdateUUID(ctx, img.ID, uuid); err != nil {
								log.Printf("Error updating UUID for image ID %d: %v", img.ID, err)
								continue
							}

							// Update status to Generate
							if err := repo.UpdateStatus(ctx, img.ID, "Generate"); err != nil {
								log.Printf("Error updating status for image ID %d: %v", img.ID, err)
								continue
							}

							log.Printf("Successfully initiated generation for image ID %d with UUID: %s", img.ID, uuid)
							continue
						}
					}

					// If it's not an INITIAL status or we couldn't extract UUID, handle as error
					log.Printf("Error generating image ID %d: %v", img.ID, err)
					if err := repo.UpdateStatus(ctx, img.ID, "Failed"); err != nil {
						log.Printf("Error updating status for image ID %d: %v", img.ID, err)
					}
					continue
				}

				// Handle successful response with UUID
				log.Printf("Image generation initiated for ID %d with UUID: %s", img.ID, resp.UUID)

				// Update image UUID
				if err := repo.UpdateUUID(ctx, img.ID, resp.UUID); err != nil {
					log.Printf("Error updating UUID for image ID %d: %v", img.ID, err)
					continue
				}

				// Update status to Generate
				if err := repo.UpdateStatus(ctx, img.ID, "Generate"); err != nil {
					log.Printf("Error updating status for image ID %d: %v", img.ID, err)
					continue
				}

				log.Printf("Successfully initiated generation for image ID %d with UUID: %s", img.ID, resp.UUID)
			}
		}
	}
}

func processGeneratedImagesWorkflow(ctx context.Context, repo repository.ImageRepository, service *service.ImageGenerationService, cfg *config.Config) {
	ticker := time.NewTicker(5 * time.Second) // Using fixed interval for now
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Image processing workflow stopped")
			return
		case <-ticker.C:
			// Get all images ready for status check
			images, err := repo.GetAllReadyToCheck(ctx)
			if err != nil {
				log.Printf("Error getting images ready for check: %v", err)
				continue
			}

			if len(images) == 0 {
				continue
			}

			for _, img := range images {
				log.Printf("Starting status checks for image ID %d with UUID: %s", img.ID, img.UUID)

				// Check status three times
				for checkCount := 1; checkCount <= 3; checkCount++ {
					log.Printf("Status check %d/3 for image ID %d with UUID: %s", checkCount, img.ID, img.UUID)

					resp, err := service.CheckGenerationStatus(ctx, img.UUID)
					if err != nil {
						// Check for 404 status in error body
						if strings.Contains(err.Error(), "\"status\":404") {
							log.Printf("API returned 404 for image ID %d, resetting UUID and status", img.ID)
							if err := repo.UpdateUUID(ctx, img.ID, ""); err != nil {
								log.Printf("Error resetting UUID for image ID %d: %v", img.ID, err)
								continue
							}
							if err := repo.UpdateStatus(ctx, img.ID, "ReadyToGenerate"); err != nil {
								log.Printf("Error updating status for image ID %d: %v", img.ID, err)
								continue
							}
							log.Printf("Image ID %d reset to ReadyToGenerate due to 404 status", img.ID)
							break // Move to next image after handling 404
						}
						log.Printf("Error getting status for image ID %d (check %d/3): %v", img.ID, checkCount, err)
						continue
					}

					log.Printf("Status for image ID %d (check %d/3): %s", img.ID, checkCount, resp.Status)

					// Handle different statuses
					switch resp.Status {
					case "DONE":
						if len(resp.Files) > 0 {
							log.Printf("Image ID %d generation completed, saving result", img.ID)
							if err := repo.UpdateBase64(ctx, img.ID, resp.Files[0]); err != nil {
								log.Printf("Error saving base64 for image ID %d: %v", img.ID, err)
								continue
							}
							if err := repo.UpdateStatus(ctx, img.ID, "ReadyToPublish"); err != nil {
								log.Printf("Error updating status for image ID %d: %v", img.ID, err)
								continue
							}
							log.Printf("Successfully saved and marked as ready to publish image ID %d", img.ID)
							break // Move to next image after successful completion
						}

					case "FAILED":
						log.Printf("Image ID %d generation failed", img.ID)
						if err := repo.UpdateStatus(ctx, img.ID, "Failed"); err != nil {
							log.Printf("Error updating status for image ID %d: %v", img.ID, err)
						}
						break // Move to next image after failure

					default:
						log.Printf("Image ID %d generation still in progress (check %d/3)", img.ID, checkCount)
						if checkCount < 3 {
							time.Sleep(2 * time.Second) // Wait 2 seconds between checks
						}
					}
				}
			}
		}
	}
}
