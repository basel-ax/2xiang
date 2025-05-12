# Image Generation Service

A Go-based service for generating images using the Fusion Brain API with PostgreSQL storage.

## Prerequisites

- Go 1.21 or later
- PostgreSQL database
- Fusion Brain API credentials

## Configuration

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` and set your configuration:
```env
# Fusion Brain API Configuration
FUSION_BRAIN_API_KEY=your-api-key-here
FUSION_BRAIN_SECRET_KEY=your-secret-key-here

# Image Generation Defaults
DEFAULT_IMAGE_WIDTH=1024
DEFAULT_IMAGE_HEIGHT=1024
DEFAULT_NUM_IMAGES=1
DEFAULT_STYLE=ANIME
DEFAULT_NEGATIVE_PROMPT=worst quality, normal quality, low quality, low res, blurry, text, watermark, logo, banner, extra digits, cropped, jpeg artifacts, signature, username, error, sketch ,duplicate, ugly, monochrome, geometry, mutation, disgusting
DEFAULT_GENERATION_TIMEOUT=300
DEFAULT_CHECK_INTERVAL=2
DEFAULT_MAX_ATTEMPTS=30

# Database Configuration
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

## Database Setup

1. Create a PostgreSQL database
2. Run the schema creation script:
```sql
CREATE TABLE IF NOT EXISTS images (
    id SERIAL PRIMARY KEY,
    prompt TEXT NOT NULL,
    uuid TEXT,
    status TEXT NOT NULL DEFAULT 'ReadyToGenerate',
    base64 TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_uuid ON images(uuid);
```

## Running the Service

The service consists of three separate workflows that can be run independently or together:

1. Image Generation Workflow (`-generator`): Handles the initial image generation requests
2. Image Processing Workflow (`-processor`): Monitors and processes generated images
3. Scheduled Workflow (`-cron`): Runs the generator every 5 minutes and processor every 10 minutes

### Command Line Options

```bash
# Run only the image generation workflow
go run cmd/example/main.go -generator

# Run only the image processing workflow
go run cmd/example/main.go -processor

# Run both workflows together
go run cmd/example/main.go -generator -processor

# Run workflows on schedule (generator every 5min, processor every 10min)
go run cmd/example/main.go -cron

# Enable verbose logging (can be combined with any workflow)
go run cmd/example/main.go -generator -verbose
```

### Workflow Descriptions

#### Image Generation Workflow (`-generator`)
- Monitors for new image requests with status 'ReadyToGenerate'
- Automatically truncates prompts longer than 999 characters while preserving UTF-8 characters
- Sends requests to the Fusion Brain API
- Updates image status to 'Generate' and saves UUID
- Handles initial API responses and errors

#### Image Processing Workflow (`-processor`)
- Monitors images with status 'Generate'
- Checks generation status with the API
- Updates image status to 'ReadyToPublish' when complete
- Handles failed generations and errors

#### Scheduled Workflow (`-cron`)
- Runs the generator workflow every 5 minutes
- Runs the processor workflow every 10 minutes
- Ensures that generator and processor jobs do not run simultaneously (mutex-based synchronization)
- Provides automated, periodic execution of both workflows
- Maintains separate schedules for generation and processing
- Logs the start and completion of each scheduled run

**Note:**
> The cron workflow uses a mutex to guarantee that only one of the scheduled jobs (generator or processor) runs at a time. If a job is still running when the next is scheduled, the new job will wait until the previous one finishes. This prevents any overlap or concurrency issues between the generator and processor workflows when scheduled by cron.

## Starting Image Generation

1. Insert a new image generation request into the database:
```sql
INSERT INTO images (prompt, status) 
VALUES ('a beautiful sunset over mountains', 'ReadyToGenerate');
```

2. Run the appropriate workflow(s) based on your needs:
```bash
# For new image generation
go run cmd/example/main.go -generator

# For processing existing generations
go run cmd/example/main.go -processor

# For both operations
go run cmd/example/main.go -generator -processor
```

## Logging

The service provides two logging modes:

1. Basic Logging (default):
   - Shows date and time for each log entry
   - Example: `2024/03/21 15:04:05 Processing image ID 1 with prompt: a beautiful sunset`

2. Verbose Logging (with `-verbose` flag):
   - Shows date, time, file name, and line number
   - Helpful for debugging and development
   - Example: `2024/03/21 15:04:05 main.go:123: Processing image ID 1 with prompt: a beautiful sunset`

Log messages include:
- Service startup and shutdown events
- Database connection status
- Image generation progress
- API request status
- Error conditions with context
- Workflow state changes

## Image Status Flow

The image generation process follows these statuses:
- `ReadyToGenerate`: Initial state when a new image request is inserted
- `Generate`: Image is being generated by the Fusion Brain API
- `ReadyToPublish`: Generation successful, base64 data is saved
- `Failed`: Generation failed

## Configuration Options

### Image Generation Defaults
- `DEFAULT_IMAGE_WIDTH`: Width of generated images (default: 1024)
- `DEFAULT_IMAGE_HEIGHT`: Height of generated images (default: 1024)
- `DEFAULT_NUM_IMAGES`: Number of images to generate per request (default: 1)
- `DEFAULT_STYLE`: Style of the generated images (default: ANIME)
- `DEFAULT_NEGATIVE_PROMPT`: Negative prompt to avoid unwanted elements
- `DEFAULT_GENERATION_TIMEOUT`: Timeout for generation in seconds (default: 300)
- `DEFAULT_CHECK_INTERVAL`: Interval between status checks in seconds (default: 2)
- `DEFAULT_MAX_ATTEMPTS`: Maximum number of status check attempts (default: 30)

### Database Configuration
- `DB_HOST`: Database host
- `DB_PORT`: Database port
- `DB_USER`: Database user
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name
- `DB_SSL_MODE`: SSL mode for database connection
- `DB_MAX_OPEN_CONNS`: Maximum number of open connections
- `DB_MAX_IDLE_CONNS`: Maximum number of idle connections
- `DB_CONN_MAX_LIFETIME`: Maximum lifetime of connections in seconds

## Project Structure

```
.
├── cmd/
│   └── example/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── domain/
│   │   └── image.go         # Domain models
│   ├── repository/
│   │   ├── image_repository.go  # Database operations
│   │   └── schema.sql       # Database schema
│   └── service/
│       └── image_service.go # Business logic
├── .env.example             # Example environment configuration
└── README.md               # This file
```

## Error Handling

The service includes comprehensive error handling:
- Database connection errors
- API request failures
- Image generation timeouts
- Invalid configurations

All errors are logged with appropriate context for debugging.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request