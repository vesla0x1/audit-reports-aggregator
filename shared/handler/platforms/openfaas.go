// shared/handler/platforms/openfaas.go
package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"shared/handler"

	"github.com/google/uuid"
)

// OpenFaaSAdapter adapts the handler for OpenFaaS functions.
// OpenFaaS uses a stdin/stdout model where the function reads from stdin
// and writes to stdout, making it simple but powerful.
type OpenFaaSAdapter struct {
	handler *handler.Handler
}

// NewOpenFaaSAdapter creates a new OpenFaaS adapter with the provided handler.
func NewOpenFaaSAdapter(h *handler.Handler) *OpenFaaSAdapter {
	return &OpenFaaSAdapter{handler: h}
}

// Handle processes a single OpenFaaS function invocation.
// This reads from stdin, processes the request, and writes to stdout.
func (a *OpenFaaSAdapter) Handle() error {
	// Read input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return a.writeError(fmt.Errorf("failed to read stdin: %w", err))
	}

	// Build request from input
	req, err := a.buildRequest(input)
	if err != nil {
		return a.writeError(fmt.Errorf("failed to build request: %w", err))
	}

	// Create context with OpenFaaS timeout
	ctx := a.createContext()

	// Process request through handler
	resp, err := a.handler.Handle(ctx, req)
	if err != nil {
		// Log error to stderr (OpenFaaS captures this)
		fmt.Fprintf(os.Stderr, "Handler error: %v\n", err)

		// Create error response
		resp = handler.NewErrorResponse(
			req.ID,
			"PROCESSING_ERROR",
			"Failed to process request",
			err.Error(),
		)
	}

	// Write response to stdout
	return a.writeResponse(resp)
}

// ServeHTTP implements http.Handler for OpenFaaS watchdog mode.
// Some OpenFaaS configurations use HTTP mode instead of stdin/stdout.
func (a *OpenFaaSAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Build request
	req, err := a.buildRequestFromHTTP(r, body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Process request
	ctx := r.Context()
	resp, err := a.handler.Handle(ctx, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(handler.NewErrorResponse(
			req.ID,
			"PROCESSING_ERROR",
			"Failed to process request",
			err.Error(),
		))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", resp.ID)

	if resp.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(resp)
}

// buildRequest creates a platform-agnostic request from OpenFaaS input
func (a *OpenFaaSAdapter) buildRequest(input []byte) (handler.Request, error) {
	// Try to parse as JSON request first
	var req handler.Request
	if err := json.Unmarshal(input, &req); err == nil {
		// Input is already a proper request object
		if req.ID == "" {
			req.ID = uuid.New().String()
		}
		if req.Timestamp.IsZero() {
			req.Timestamp = time.Now().UTC()
		}
		return req, nil
	}

	// Otherwise treat as raw payload
	metadata := a.extractMetadataFromEnv()

	return handler.Request{
		ID:        metadata["request_id"],
		Source:    "openfaas",
		Type:      a.extractRequestType(),
		Payload:   json.RawMessage(input),
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}, nil
}

// buildRequestFromHTTP creates a request from HTTP mode invocation
func (a *OpenFaaSAdapter) buildRequestFromHTTP(r *http.Request, body []byte) (handler.Request, error) {
	// Extract metadata from headers
	metadata := make(map[string]string)

	// OpenFaaS specific headers
	if funcName := r.Header.Get("X-Function-Name"); funcName != "" {
		metadata["function_name"] = funcName
	}

	// Request ID
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = r.Header.Get("X-Call-Id")
	}
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Request type
	requestType := r.Header.Get("X-Request-Type")
	if requestType == "" {
		requestType = os.Getenv("OPENFAAS_FUNCTION_NAME")
		if requestType == "" {
			requestType = "function"
		}
	}

	// Add other relevant headers
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-") && len(values) > 0 {
			metadata[strings.ToLower(strings.ReplaceAll(key, "-", "_"))] = values[0]
		}
	}

	return handler.Request{
		ID:        requestID,
		Source:    "openfaas",
		Type:      requestType,
		Payload:   json.RawMessage(body),
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}, nil
}

// extractMetadataFromEnv extracts metadata from OpenFaaS environment variables
func (a *OpenFaaSAdapter) extractMetadataFromEnv() map[string]string {
	metadata := make(map[string]string)

	// OpenFaaS environment variables
	envVars := map[string]string{
		"OPENFAAS_FUNCTION_NAME": "function_name",
		"OPENFAAS_NAMESPACE":     "namespace",
		"HOSTNAME":               "hostname",
		"Http_Path":              "http_path",
		"Http_Method":            "http_method",
		"Http_Query":             "http_query",
		"Http_ContentType":       "content_type",
	}

	for envKey, metaKey := range envVars {
		if value := os.Getenv(envKey); value != "" {
			metadata[metaKey] = value
		}
	}

	// Extract HTTP headers from environment
	// OpenFaaS passes HTTP headers as Http_ prefixed env vars
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "Http_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimPrefix(parts[0], "Http_"))
				metadata["header_"+key] = parts[1]
			}
		}
	}

	// Generate request ID if not present
	if metadata["request_id"] == "" {
		if reqID := os.Getenv("Http_X_Request_Id"); reqID != "" {
			metadata["request_id"] = reqID
		} else {
			metadata["request_id"] = uuid.New().String()
		}
	}

	return metadata
}

// extractRequestType determines the request type
func (a *OpenFaaSAdapter) extractRequestType() string {
	// Check for explicit type in environment
	if reqType := os.Getenv("Http_X_Request_Type"); reqType != "" {
		return reqType
	}

	// Use function name as type
	if funcName := os.Getenv("OPENFAAS_FUNCTION_NAME"); funcName != "" {
		return funcName
	}

	// Check HTTP path
	if path := os.Getenv("Http_Path"); path != "" {
		path = strings.TrimPrefix(path, "/")
		if path != "" {
			return path
		}
	}

	return "function"
}

// createContext creates a context with OpenFaaS timeout
func (a *OpenFaaSAdapter) createContext() context.Context {
	ctx := context.Background()

	// Add OpenFaaS metadata to context
	ctx = context.WithValue(ctx, "function_name", os.Getenv("OPENFAAS_FUNCTION_NAME"))
	ctx = context.WithValue(ctx, "namespace", os.Getenv("OPENFAAS_NAMESPACE"))

	// OpenFaaS timeout is typically handled by the watchdog
	// but we can add our own timeout if specified
	if timeoutStr := os.Getenv("OPENFAAS_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			ctx, _ = context.WithTimeout(ctx, timeout)
		}
	}

	return ctx
}

// writeResponse writes the response to stdout for OpenFaaS
func (a *OpenFaaSAdapter) writeResponse(resp handler.Response) error {
	// Marshal response to JSON
	output, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Write to stdout
	_, err = fmt.Fprint(os.Stdout, string(output))
	return err
}

// writeError writes an error response
func (a *OpenFaaSAdapter) writeError(err error) error {
	// Log to stderr for debugging
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)

	// Create error response
	resp := handler.NewErrorResponse(
		uuid.New().String(),
		"ADAPTER_ERROR",
		"Failed to process request",
		err.Error(),
	)

	// Write to stdout
	return a.writeResponse(resp)
}

// Run is the main entry point for OpenFaaS functions
// This should be called from the main function
func (a *OpenFaaSAdapter) Run() {
	// Check if we're in HTTP mode
	if os.Getenv("OPENFAAS_HTTP") == "true" {
		// Start HTTP server
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		fmt.Fprintf(os.Stderr, "Starting HTTP server on port %s\n", port)
		if err := http.ListenAndServe(":"+port, a); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// stdin/stdout mode
		if err := a.Handle(); err != nil {
			fmt.Fprintf(os.Stderr, "Handler error: %v\n", err)
			os.Exit(1)
		}
	}
}
