# 2Xiang Image Generation Service

This service provides a clean, idiomatic Go implementation for generating images using the Fusion Brain API.

## Features

- Clean architecture implementation
- Full support for Fusion Brain API features
- Configurable image generation parameters
- Robust error handling and timeout management
- Support for style presets and negative prompts
- Automatic polling for generation status
- Environment-based configuration
- PostgreSQL database integration

## Installation

```bash
go get github.com/basel-ax/2xiang
```

## Configuration

Create a `.env` file in your project root with the following variables:

```env
# Fusion Brain API Configuration
FUSION_BRAIN_API_KEY=your-api-key-here
FUSION_BRAIN_SECRET_KEY=your-secret-key-here

# Image Generation Defaults
DEFAULT_IMAGE_WIDTH=1024
DEFAULT_IMAGE_HEIGHT=1024
DEFAULT_NUM_IMAGES=1
DEFAULT_GENERATION_TIMEOUT=300 # 5 minutes in seconds
DEFAULT_CHECK_INTERVAL=2 # seconds
DEFAULT_MAX_ATTEMPTS=30

# PostgreSQL Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password-here
DB_NAME=your-database-name
DB_SSL_MODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME=300 # 5 minutes in seconds
```

## Usage

1. Set up your `.env` file with the required configuration (see above).

2. Use the service in your code:

```go
import (
    "context"
    "time"
    
    "github.com/basel-ax/2xiang/internal/config"
    "github.com/basel-ax/2xiang/internal/domain"
    "github.com/basel-ax/2xiang/internal/service"
)

// Load configuration
cfg, err := config.Load()
if err != nil {
    // Handle error
}

// Create the service
imgService := service.NewImageGenerationService(cfg)

// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), cfg.GenerationTimeout)
defer cancel()

// Create the image generation request
req := domain.ImageGenerationRequest{
    Prompt:         "Your prompt here",
    Width:          cfg.DefaultImageWidth,
    Height:         cfg.DefaultImageHeight,
    NumImages:      cfg.DefaultNumImages,
    Style:          "ANIME", // Optional
    NegativePrompt: "blurry, low quality", // Optional
}

// Generate the image
resp, err := imgService.GenerateImage(ctx, req)
if err != nil {
    // Handle error
}

// Wait for the generation to complete
finalResp, err := imgService.WaitForGeneration(ctx, resp.UUID)
if err != nil {
    // Handle error
}

// Use the generated image URLs
for _, file := range finalResp.Files {
    // Process the image URL
}
```

## Configuration Options

The service supports the following configuration options in the `.env` file:

### Fusion Brain API Configuration
- `FUSION_BRAIN_API_KEY`: Your Fusion Brain API key (required)
- `FUSION_BRAIN_SECRET_KEY`: Your Fusion Brain secret key (required)

### Image Generation Defaults
- `DEFAULT_IMAGE_WIDTH`: Default image width (default: 1024)
- `DEFAULT_IMAGE_HEIGHT`: Default image height (default: 1024)
- `DEFAULT_NUM_IMAGES`: Default number of images to generate (default: 1)
- `DEFAULT_GENERATION_TIMEOUT`: Timeout for generation in seconds (default: 300)
- `DEFAULT_CHECK_INTERVAL`: Interval between status checks in seconds (default: 2)
- `DEFAULT_MAX_ATTEMPTS`: Maximum number of status check attempts (default: 30)

### PostgreSQL Database Configuration
- `DB_HOST`: Database host (required)
- `DB_PORT`: Database port (default: 5432)
- `DB_USER`: Database user (required)
- `DB_PASSWORD`: Database password (required)
- `DB_NAME`: Database name (required)
- `DB_SSL_MODE`: SSL mode (default: disable)
- `DB_MAX_OPEN_CONNS`: Maximum number of open connections (default: 25)
- `DB_MAX_IDLE_CONNS`: Maximum number of idle connections (default: 25)
- `DB_CONN_MAX_LIFETIME`: Connection maximum lifetime in seconds (default: 300)

## Error Handling

The service provides detailed error messages and proper error wrapping. All errors are returned with context about what went wrong.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.