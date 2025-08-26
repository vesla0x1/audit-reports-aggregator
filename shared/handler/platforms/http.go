package platforms

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"shared/handler"

	"github.com/google/uuid"
)

// HTTPAdapter adapts the handler for standard HTTP servers.
// This adapter can be used for local development, Kubernetes deployments,
// or any standard HTTP server environment.
type HTTPAdapter struct {
	handler *handler.Handler
}

// NewHTTPAdapter creates a new HTTP adapter with the provided handler.
func NewHTTPAdapter(h *handler.Handler) *HTTPAdapter {
	return &HTTPAdapter{handler: h}
}

// ServeHTTP implements the http.Handler interface, allowing the adapter
// to be used with any standard HTTP server or router.
func (a *HTTPAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle health check endpoints
	if a.isHealthCheck(r.URL.Path) {
		a.handleHealth(w, r)
		return
	}

	// Handle metrics endpoint (if needed)
	if r.URL.Path == "/metrics" {
		// Metrics are typically handled by Prometheus handler separately
		http.Error(w, "Metrics endpoint should be configured separately", http.StatusNotFound)
		return
	}

	// Read and validate request body
	body, err := a.readBody(r)
	if err != nil {
		a.writeErrorResponse(w, handler.NewErrorResponse(
			uuid.New().String(),
			"INVALID_REQUEST",
			"Failed to read request body",
			err.Error(),
		))
		return
	}

	// Build platform-agnostic request
	req := a.buildRequest(r, body)

	// Process request through handler
	ctx := r.Context()
	resp, err := a.handler.Handle(ctx, req)

	// Write response
	a.writeResponse(w, resp, err)
}

// isHealthCheck checks if the path is a health check endpoint
func (a *HTTPAdapter) isHealthCheck(path string) bool {
	healthPaths := []string{
		"/health",
		"/healthz",
		"/ready",
		"/readyz",
		"/live",
		"/livez",
	}

	for _, healthPath := range healthPaths {
		if path == healthPath {
			return true
		}
	}
	return false
}

// handleHealth handles health check requests
func (a *HTTPAdapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check handler health
	if err := a.handler.Health(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	// Return healthy status
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"worker": a.handler.Worker().Name(),
		"time":   time.Now().UTC(),
	})
}

// readBody reads and validates the request body
func (a *HTTPAdapter) readBody(r *http.Request) ([]byte, error) {
	// Get max request size from handler config
	maxSize := a.handler.Config().MaxRequestSize
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // Default 10MB
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return body, nil
}

// buildRequest creates a platform-agnostic request from HTTP request
func (a *HTTPAdapter) buildRequest(r *http.Request, body []byte) handler.Request {
	// Extract request ID from headers
	requestID := a.extractRequestID(r)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Extract request type
	requestType := a.extractRequestType(r)

	// Build metadata from headers and request info
	metadata := a.extractMetadata(r)

	return handler.Request{
		ID:        requestID,
		Source:    "http",
		Type:      requestType,
		Payload:   json.RawMessage(body),
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}
}

// extractRequestID attempts to extract request ID from headers
func (a *HTTPAdapter) extractRequestID(r *http.Request) string {
	// Common request ID headers
	headers := []string{
		"X-Request-ID",
		"X-Request-Id",
		"X-Correlation-ID",
		"X-Correlation-Id",
		"Request-ID",
		"Request-Id",
	}

	for _, header := range headers {
		if id := r.Header.Get(header); id != "" {
			return id
		}
	}

	return ""
}

// extractRequestType determines the request type from the HTTP request
func (a *HTTPAdapter) extractRequestType(r *http.Request) string {
	// First check for explicit type header
	if reqType := r.Header.Get("X-Request-Type"); reqType != "" {
		return reqType
	}

	// Try to extract from path
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" && path != "/" {
		// Take the first path segment as type
		if idx := strings.Index(path, "/"); idx > 0 {
			return path[:idx]
		}
		return path
	}

	// Default to HTTP method
	return strings.ToLower(r.Method)
}

// extractMetadata builds metadata from HTTP request
func (a *HTTPAdapter) extractMetadata(r *http.Request) map[string]string {
	metadata := make(map[string]string)

	// Add request info
	metadata["http_method"] = r.Method
	metadata["http_path"] = r.URL.Path
	metadata["http_host"] = r.Host

	// Add query parameters
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			metadata["query_"+key] = values[0]
		}
	}

	// Add selected headers
	relevantHeaders := []string{
		"Content-Type",
		"Accept",
		"User-Agent",
		"X-Forwarded-For",
		"X-Real-IP",
		"Authorization",
	}

	for _, header := range relevantHeaders {
		if value := r.Header.Get(header); value != "" {
			// Sanitize authorization header
			if header == "Authorization" {
				if strings.HasPrefix(value, "Bearer ") {
					value = "Bearer [REDACTED]"
				} else {
					value = "[REDACTED]"
				}
			}
			metadata["header_"+strings.ToLower(strings.ReplaceAll(header, "-", "_"))] = value
		}
	}

	// Add trace headers if present
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		metadata["trace_id"] = traceID
	}

	return metadata
}

// writeResponse writes the handler response as HTTP response
func (a *HTTPAdapter) writeResponse(w http.ResponseWriter, resp handler.Response, err error) {
	// Set common headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", resp.ID)

	// Add response metadata as headers
	for key, value := range resp.Metadata {
		w.Header().Set("X-"+key, value)
	}

	// Handle processing error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(handler.NewErrorResponse(
			resp.ID,
			"INTERNAL_ERROR",
			"Request processing failed",
			err.Error(),
		))
		return
	}

	// Determine status code based on response
	statusCode := a.determineStatusCode(resp)
	w.WriteHeader(statusCode)

	// Write response body
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Log error but response headers are already sent
		// This should be logged via observability
	}
}

// writeErrorResponse writes an error response
func (a *HTTPAdapter) writeErrorResponse(w http.ResponseWriter, resp handler.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", resp.ID)

	statusCode := a.determineStatusCode(resp)
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(resp)
}

// determineStatusCode maps response to HTTP status code
func (a *HTTPAdapter) determineStatusCode(resp handler.Response) int {
	if resp.Success {
		return http.StatusOK
	}

	if resp.Error == nil {
		return http.StatusInternalServerError
	}

	// Map error codes to HTTP status codes
	switch resp.Error.Code {
	case "VALIDATION_ERROR":
		return http.StatusBadRequest
	case "INVALID_REQUEST":
		return http.StatusBadRequest
	case "NOT_FOUND":
		return http.StatusNotFound
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "RATE_LIMITED":
		return http.StatusTooManyRequests
	case "TIMEOUT":
		return http.StatusGatewayTimeout
	case "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable
	case "INTERNAL_ERROR":
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Serve starts an HTTP server with the adapter
// This is a convenience method for quick setup
func (a *HTTPAdapter) Serve(addr string) error {
	return http.ListenAndServe(addr, a)
}
