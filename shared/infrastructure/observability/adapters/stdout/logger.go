package stdout

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"shared/domain/observability"
)

// Logger implements observability.Logger using stdout
type Logger struct {
	fields map[string]interface{}
	logger *log.Logger
}

// NewLogger creates a new stdout logger
func NewLogger() observability.Logger {
	return &Logger{
		fields: make(map[string]interface{}),
		logger: log.New(os.Stdout, "", 0), // No prefix, we'll format ourselves
	}
}

// Info logs informational messages
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

// Error logs error messages
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

// Warn logs warning messages (if you need it)
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

// Debug logs debug messages (if you need it)
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log("DEBUG", msg, fields...)
}

// WithFields returns a new Logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) observability.Logger {
	// Create new logger with combined fields
	newFields := make(map[string]interface{})

	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		fields: newFields,
		logger: l.logger,
	}
}

// log is the internal logging method
func (l *Logger) log(level string, msg string, fields ...interface{}) {
	entry := l.createLogEntry(level, msg, fields...)

	// Format as JSON for structured logging
	if jsonOutput {
		l.logJSON(entry)
	} else {
		l.logText(entry)
	}
}

// createLogEntry builds the log entry
func (l *Logger) createLogEntry(level string, msg string, fields ...interface{}) map[string]interface{} {
	entry := make(map[string]interface{})

	// Add standard fields
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	entry["level"] = level
	entry["message"] = msg

	// Add persistent fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Parse variadic fields (key1, value1, key2, value2, ...)
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		// Special handling for error type
		if key == "error" && fields[i+1] != nil {
			if err, ok := fields[i+1].(error); ok {
				entry[key] = err.Error()
			} else {
				entry[key] = fields[i+1]
			}
		} else {
			entry[key] = fields[i+1]
		}
	}

	return entry
}

// logJSON outputs the entry as JSON
func (l *Logger) logJSON(entry map[string]interface{}) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}
	l.logger.Println(string(jsonBytes))
}

// logText outputs the entry as formatted text
func (l *Logger) logText(entry map[string]interface{}) {
	// Build formatted string
	timestamp := entry["timestamp"]
	level := entry["level"]
	message := entry["message"]
	delete(entry, "timestamp")
	delete(entry, "level")
	delete(entry, "message")

	// Format additional fields
	var fieldStrs []string
	for k, v := range entry {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
	}

	// Build final log line
	logLine := fmt.Sprintf("%s [%s] %s", timestamp, level, message)
	if len(fieldStrs) > 0 {
		logLine += " | " + strings.Join(fieldStrs, " ")
	}

	l.logger.Println(logLine)
}

// Configuration

var jsonOutput = false

// UseJSON enables JSON output format
func UseJSON(enabled bool) {
	jsonOutput = enabled
}
