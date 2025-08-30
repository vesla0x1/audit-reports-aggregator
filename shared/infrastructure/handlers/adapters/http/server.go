package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"shared/config"
	"shared/domain/handler"
)

// Adapter handles HTTP server runtime integration
type Adapter struct {
	handler handler.Handler
	config  *config.HTTPConfig
	server  *http.Server
}

// NewAdapter creates a new HTTP adapter
func NewAdapter(h handler.Handler, cfg *config.HTTPConfig) *Adapter {
	return &Adapter{
		handler: h,
		config:  cfg,
	}
}

// Start begins the HTTP server
func (a *Adapter) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleRequest)

	a.server = &http.Server{
		Addr:         a.config.Addr,
		Handler:      mux,
		ReadTimeout:  a.config.Timeout,
		WriteTimeout: a.config.Timeout,
	}

	a.logStartup()

	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the HTTP server
func (a *Adapter) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	a.handler.Logger().Info("Shutting down HTTP server")
	return a.server.Shutdown(ctx)
}

// handleRequest processes incoming HTTP requests
func (a *Adapter) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		a.handleMethodNotAllowed(w, r)
		return
	}

	// Track request
	invocation := a.trackRequest(r)
	defer invocation.recordDuration()

	// Read body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		a.handleBadRequest(w, r, fmt.Errorf("failed to read request body: %w", err))
		return
	}

	// Parse request
	var req handler.Request
	if err := json.Unmarshal(body, &req); err != nil {
		a.handleBadRequest(w, r, fmt.Errorf("invalid JSON payload: %w", err))
		return
	}

	// Add HTTP metadata
	a.enrichRequest(&req, r)

	// Apply timeout if configured
	ctx := r.Context()
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// Process request
	resp, err := a.handler.Handle(ctx, req)

	// Send response
	a.sendResponse(w, resp, err)

	// Log result
	a.logRequestComplete(req, resp, err)
}

// enrichRequest adds HTTP-specific metadata to the request
func (a *Adapter) enrichRequest(req *handler.Request, r *http.Request) {
	// Set defaults if not provided
	if req.ID == "" {
		req.ID = a.generateRequestID()
	}
	if req.Source == "" {
		req.Source = "http"
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// Add HTTP metadata
	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}
	req.Metadata["http_method"] = r.Method
	req.Metadata["http_path"] = r.URL.Path
	req.Metadata["http_remote_addr"] = r.RemoteAddr

	// Add select headers
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		req.Metadata["http_content_type"] = contentType
	}
	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		req.Metadata["http_user_agent"] = userAgent
	}
}

// sendResponse writes the handler response to the HTTP response
func (a *Adapter) sendResponse(w http.ResponseWriter, resp handler.Response, err error) {
	w.Header().Set("Content-Type", "application/json")

	// Handle error case
	if err != nil {
		a.sendErrorResponse(w, err)
		return
	}

	// Determine status code
	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = http.StatusUnprocessableEntity
	}

	// Write response
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		a.handler.Logger().Error("Failed to encode response", "error", err)
	}
}

// sendErrorResponse sends an error response
func (a *Adapter) sendErrorResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	resp := handler.Response{
		Success: false,
		Error:   &handler.ErrorInfo{},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		a.handler.Logger().Error("Failed to encode error response", "error", err)
	}
}

// handleMethodNotAllowed handles non-POST requests
func (a *Adapter) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	a.handler.Logger().Info("Method not allowed",
		"method", r.Method,
		"path", r.URL.Path)
	a.handler.Metrics().IncrementCounter("http.method_not_allowed", nil)

	w.Header().Set("Allow", "POST")
	http.Error(w, "Method not allowed. Only POST is supported.", http.StatusMethodNotAllowed)
}

// handleBadRequest handles invalid requests
func (a *Adapter) handleBadRequest(w http.ResponseWriter, r *http.Request, err error) {
	a.handler.Logger().Error("Bad request", "error", err)
	a.handler.Metrics().IncrementCounter("http.bad_request", nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	resp := handler.Response{
		Success: false,
		Error:   &handler.ErrorInfo{},
	}

	json.NewEncoder(w).Encode(resp)
}

// generateRequestID creates a unique request ID
func (a *Adapter) generateRequestID() string {
	return fmt.Sprintf("http-%d", time.Now().UnixNano())
}

// --- Logging Helpers ---

func (a *Adapter) logStartup() {
	a.handler.Logger().Info("Starting HTTP adapter", "address", a.config.Addr)
	a.handler.Metrics().IncrementCounter("http.starts", nil)
}

func (a *Adapter) logRequestComplete(req handler.Request, resp handler.Response, err error) {
	logger := a.handler.Logger()

	if err != nil {
		logger.Error("Request processing failed",
			"request_id", req.ID,
			"error", err)
	} else if !resp.Success {
		logger.Info("Request processing unsuccessful",
			"request_id", req.ID,
			"error", resp.Error)
	} else {
		logger.Info("Request processed successfully",
			"request_id", req.ID)
	}
}

// --- Metrics Helpers ---

type requestTracker struct {
	adapter   *Adapter
	startTime time.Time
}

func (a *Adapter) trackRequest(r *http.Request) *requestTracker {
	a.handler.Logger().Info("HTTP request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)
	a.handler.Metrics().IncrementCounter("http.requests", nil)

	return &requestTracker{
		adapter:   a,
		startTime: time.Now(),
	}
}

func (t *requestTracker) recordDuration() {
	duration := time.Since(t.startTime)
	t.adapter.handler.Metrics().RecordHistogram("http.request_duration",
		float64(duration.Milliseconds()), nil)
}
