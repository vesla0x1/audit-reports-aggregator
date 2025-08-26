/*
Package downloader provides functionality for downloading audit reports and documents
from various security audit providers.

The downloader worker is responsible for:
  - Downloading HTML pages and PDF documents from audit provider URLs
  - Validating and processing the downloaded content
  - Storing content in object storage (MinIO)
  - Publishing events when downloads complete or fail
  - Providing HTTP endpoints for triggering downloads

Architecture

The downloader follows Clean Architecture principles with clear separation of concerns:

	├── cmd/                    # Application entry point
	├── internal/
	│   ├── domain/            # Business entities and logic
	│   ├── service/           # Application services (use cases)
	│   ├── handler/           # HTTP handlers (delivery layer)
	│   ├── ports/             # Interface definitions
	│   └── adapters/          # External adapters
	│       ├── http/          # HTTP client adapter
	│       ├── storage/       # MinIO storage adapter
	│       └── events/        # Event publishing adapter

Domain Model

The core domain revolves around the Download entity, which represents a download operation:

	type Download struct {
	    ReportID  string       // Unique identifier for the report
	    SourceID  string       // Source provider identifier
	    URL       string       # URL to download from
	    Type      DownloadType # Type of content (HTML or Document)
	    Content   []byte       # Downloaded content
	    Hash      string       # SHA256 hash of content
	    // ... additional metadata
	}

Supported content types:
  - HTML pages from audit provider websites
  - PDF documents containing audit reports
  - Markdown files (detected via MIME type)

Usage

The service exposes an HTTP endpoint that accepts download requests:

	POST /
	Content-Type: application/json

	{
	    "report_id": "audit-123",
	    "source_id": "provider-xyz",
	    "url": "https://provider.com/report.pdf",
	    "type": "document",
	    "title": "Security Audit Report"
	}

The response contains information about the stored file:

	{
	    "success": true,
	    "report_id": "audit-123",
	    "minio_key": "provider-xyz/2024-01-15/audit-123_a1b2c3d4.pdf",
	    "file_size": 1024000,
	    "file_hash": "sha256hash..."
	}

Error Handling

The service implements comprehensive error handling:
  - Validation errors for invalid requests
  - Network errors during download
  - Storage errors when saving content
  - Content validation errors (empty content, file too large)

All errors are logged with structured logging and metrics are recorded for monitoring.

Configuration

The service requires the following configuration:
  - HTTP timeout settings
  - MinIO storage configuration (endpoint, credentials, bucket)
  - Metrics port for Prometheus monitoring
  - Logging configuration

Events

The service publishes events for integration with other workers:
  - DownloadCompleted: Published when a download succeeds
  - DownloadFailed: Published when a download fails

Observability

The service includes comprehensive observability:
  - Structured logging with zerolog
  - Prometheus metrics for monitoring
  - Request tracing and timing
  - Health check endpoints

Security

Security considerations:
  - File size limits to prevent resource exhaustion
  - Content type validation
  - Secure storage with MinIO
  - No sensitive data in logs
*/
package main