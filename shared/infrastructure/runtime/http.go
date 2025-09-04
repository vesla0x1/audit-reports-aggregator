package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// handles HTTP server runtime integration
type httpRuntime struct {
	handler ports.Handler
	logger  ports.Logger
	metrics ports.Metrics
	config  *config.HTTPConfig
	server  *http.Server
}

// NewAdapter creates a new HTTP adapter
func NewHTTPRuntime(cfg *config.HTTPConfig, handler ports.Handler, obs ports.Observability) ports.Runtime {
	logger, metrics, err := obs.ComponentsScoped("runtime.http")
	if err != nil {
		fmt.Errorf("failed to create runtime: Obervability was not initialized %w", err)
	}

	if handler == nil {
		panic(fmt.Errorf("failed to create runtime: handle is required"))
	}

	if handler == nil {
		fmt.Errorf("failed to create runtime: handle is required")
	}

	return &httpRuntime{
		logger:  logger,
		metrics: metrics,
		handler: handler,
		config:  cfg,
	}
}

// Start begins the HTTP server
func (httpRuntime *httpRuntime) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpRuntime.handleRequest)

	httpRuntime.server = &http.Server{
		Addr:         httpRuntime.config.Addr,
		Handler:      mux,
		ReadTimeout:  httpRuntime.config.Timeout,
		WriteTimeout: httpRuntime.config.Timeout,
	}

	httpRuntime.logStartup()

	if err := httpRuntime.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the HTTP server
func (httpRuntime *httpRuntime) Stop(ctx context.Context) error {
	if httpRuntime.server == nil {
		return nil
	}

	httpRuntime.logger.Info("Shutting down HTTP server")
	return httpRuntime.server.Shutdown(ctx)
}

// handleRequest processes incoming HTTP requests
func (httpRuntime *httpRuntime) handleRequest(resWriter http.ResponseWriter, request *http.Request) {
	// Only accept POST requests
	if request.Method != http.MethodPost {
		httpRuntime.handleMethodNotAllowed(resWriter, request)
		return
	}

	// Track request
	invocation := httpRuntime.trackRequest(request)
	defer invocation.recordDuration()

	// Read body
	body, err := io.ReadAll(request.Body)
	defer request.Body.Close()

	if err != nil {
		httpRuntime.handleBadRequest(resWriter, request, fmt.Errorf("failed to read request body: %w", err))
		return
	}

	// Parse request
	var httpRuntimeReq ports.RuntimeRequest
	if err := json.Unmarshal(body, &httpRuntimeReq); err != nil {
		httpRuntime.handleBadRequest(resWriter, request, fmt.Errorf("invalid JSON payload: %w", err))
		return
	}

	// Add HTTP metadata
	httpRuntime.enrichRequest(&httpRuntimeReq, request)

	// Apply timeout if configured
	ctx := request.Context()
	if httpRuntime.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, httpRuntime.config.Timeout)
		defer cancel()
	}

	// Process request
	resp, err := httpRuntime.handler.Handle(ctx, httpRuntimeReq)

	// Send response
	httpRuntime.sendResponse(resWriter, resp, err)

	// Log result
	httpRuntime.logRequestComplete(httpRuntimeReq, resp, err)
}

// enrichRequest adds HTTP-specific metadata to the request
func (httpRuntime *httpRuntime) enrichRequest(httpRuntimeReq *ports.RuntimeRequest, httpReq *http.Request) {
	// Set defaults if not provided
	if httpRuntimeReq.ID == "" {
		httpRuntimeReq.ID = httpRuntime.generateRequestID()
	}
	if httpRuntimeReq.Source == "" {
		httpRuntimeReq.Source = "http"
	}
	if httpRuntimeReq.Timestamp.IsZero() {
		httpRuntimeReq.Timestamp = time.Now().UTC()
	}

	// Add HTTP metadata
	if httpRuntimeReq.Metadata == nil {
		httpRuntimeReq.Metadata = make(map[string]string)
	}
	httpRuntimeReq.Metadata["http_method"] = httpReq.Method
	httpRuntimeReq.Metadata["http_path"] = httpReq.URL.Path
	httpRuntimeReq.Metadata["http_remote_addr"] = httpReq.RemoteAddr

	// Add select headers
	if contentType := httpReq.Header.Get("Content-Type"); contentType != "" {
		httpRuntimeReq.Metadata["http_content_type"] = contentType
	}
	if userAgent := httpReq.Header.Get("User-Agent"); userAgent != "" {
		httpRuntimeReq.Metadata["http_user_agent"] = userAgent
	}
}

// sendResponse writes the handler response to the HTTP response
func (httpRuntime *httpRuntime) sendResponse(resWriter http.ResponseWriter, httpRuntimeResponse ports.RuntimeResponse, err error) {
	resWriter.Header().Set("Content-Type", "application/json")

	// Handle error case
	if err != nil {
		httpRuntime.sendErrorResponse(resWriter, err)
		return
	}

	// Determine status code
	statusCode := http.StatusOK
	if !httpRuntimeResponse.Success {
		statusCode = http.StatusUnprocessableEntity
	}

	// Write response
	resWriter.WriteHeader(statusCode)
	if err := json.NewEncoder(resWriter).Encode(httpRuntimeResponse); err != nil {
		httpRuntime.logger.Error("Failed to encode response", "error", err)
	}
}

// sendErrorResponse sends an error response
func (httpRuntime *httpRuntime) sendErrorResponse(resWriter http.ResponseWriter, err error) {
	resWriter.WriteHeader(http.StatusInternalServerError)

	resp := ports.RuntimeResponse{
		Success: false,
		Error:   err.Error(),
	}

	if err := json.NewEncoder(resWriter).Encode(resp); err != nil {
		httpRuntime.logger.Error("Failed to encode error response", "error", err)
	}
}

// handleMethodNotAllowed handles non-POST requests
func (httpRuntime *httpRuntime) handleMethodNotAllowed(resWriter http.ResponseWriter, req *http.Request) {
	httpRuntime.logger.Info("Method not allowed",
		"method", req.Method,
		"path", req.URL.Path)
	httpRuntime.metrics.IncrementCounter("http.method_not_allowed", nil)

	resWriter.Header().Set("Allow", "POST")
	http.Error(resWriter, "Method not allowed. Only POST is supported.", http.StatusMethodNotAllowed)
}

// handleBadRequest handles invalid requests
func (httpRuntime *httpRuntime) handleBadRequest(resWriter http.ResponseWriter, r *http.Request, err error) {
	httpRuntime.logger.Error("Bad request", "error", err)
	httpRuntime.metrics.IncrementCounter("http.bad_request", nil)

	resWriter.Header().Set("Content-Type", "application/json")
	resWriter.WriteHeader(http.StatusBadRequest)

	resp := ports.RuntimeResponse{
		Success: false,
	}

	json.NewEncoder(resWriter).Encode(resp)
}

// generateRequestID creates a unique request ID
func (httpRuntime *httpRuntime) generateRequestID() string {
	return fmt.Sprintf("http-%d", time.Now().UnixNano())
}

// --- Logging Helpers ---

func (httpRuntime *httpRuntime) logStartup() {
	httpRuntime.logger.Info("Starting HTTP adapter", "address", httpRuntime.config.Addr)
	httpRuntime.metrics.IncrementCounter("http.starts", nil)
}

func (httpRuntime *httpRuntime) logRequestComplete(req ports.RuntimeRequest, resp ports.RuntimeResponse, err error) {
	logger := httpRuntime.logger

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
	httpRuntime *httpRuntime
	startTime   time.Time
}

func (httpRuntime *httpRuntime) trackRequest(request *http.Request) *requestTracker {
	httpRuntime.logger.Info("HTTP request received",
		"method", request.Method,
		"path", request.URL.Path,
		"remote_addr", request.RemoteAddr)
	httpRuntime.metrics.IncrementCounter("http.requests", nil)

	return &requestTracker{
		httpRuntime: httpRuntime,
		startTime:   time.Now(),
	}
}

func (t *requestTracker) recordDuration() {
	duration := time.Since(t.startTime)
	t.httpRuntime.metrics.RecordHistogram("http.request_duration",
		float64(duration.Milliseconds()), nil)
}
