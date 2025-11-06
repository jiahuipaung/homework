package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ServerConfig struct {
	Port        int
	EnableAuth  bool
	Username    string
	Password    string
	EnableFault bool
	Verbose     bool
}

type MockServer struct {
	config        *ServerConfig
	server        *http.Server
	mu            sync.RWMutex
	logs          []LogEntry
	stats         Stats
	healthStatus  int32 // 0=healthy, 1=unhealthy
	faultMode     int32 // 0=normal, 1=error, 2=slow
	requestCount  int64
	errorCount    int64
	successCount  int64
	totalBytes    int64
	startTime     time.Time
}

type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
	RawData    string                 `json:"raw_data"`
	StreamFields []string             `json:"stream_fields,omitempty"`
	TimeField    string               `json:"time_field,omitempty"`
	MsgField     string               `json:"msg_field,omitempty"`
}

type Stats struct {
	TotalRequests  int64     `json:"total_requests"`
	SuccessCount   int64     `json:"success_count"`
	ErrorCount     int64     `json:"error_count"`
	TotalLogs      int       `json:"total_logs"`
	TotalBytes     int64     `json:"total_bytes"`
	Uptime         string    `json:"uptime"`
	StartTime      time.Time `json:"start_time"`
	HealthStatus   string    `json:"health_status"`
	FaultMode      string    `json:"fault_mode"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func NewMockServer(config *ServerConfig) *MockServer {
	return &MockServer{
		config:       config,
		logs:         make([]LogEntry, 0),
		startTime:    time.Now(),
		healthStatus: 0, // healthy by default
		faultMode:    0, // normal mode
	}
}

func (s *MockServer) Start() error {
	mux := http.NewServeMux()

	// VictoriaLogs endpoints
	mux.HandleFunc("/insert/jsonline", s.authMiddleware(s.handleInsertJSONLine))
	mux.HandleFunc("/health", s.handleHealth)

	// Management endpoints
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/logs", s.handleGetLogs)
	mux.HandleFunc("/clear", s.handleClear)
	mux.HandleFunc("/control/health", s.handleControlHealth)
	mux.HandleFunc("/control/fault", s.handleControlFault)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}

	return s.server.ListenAndServe()
}

func (s *MockServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

// authMiddleware checks basic authentication if enabled
func (s *MockServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableAuth {
			username, password, ok := r.BasicAuth()
			if !ok || username != s.config.Username || password != s.config.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="VictoriaLogs Mock"`)
				s.sendError(w, http.StatusUnauthorized, "Authentication required")
				atomic.AddInt64(&s.errorCount, 1)
				return
			}
		}
		next(w, r)
	}
}

// handleInsertJSONLine handles POST /insert/jsonline
func (s *MockServer) handleInsertJSONLine(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&s.requestCount, 1)

	// Check method
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		atomic.AddInt64(&s.errorCount, 1)
		return
	}

	// Apply fault injection
	if s.config.EnableFault {
		if s.handleFaultInjection(w) {
			atomic.AddInt64(&s.errorCount, 1)
			return
		}
	}

	// Check health status
	if atomic.LoadInt32(&s.healthStatus) == 1 {
		s.sendError(w, http.StatusServiceUnavailable, "Service is unhealthy")
		atomic.AddInt64(&s.errorCount, 1)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	streamFields := parseCommaSeparated(query.Get("_stream_fields"))
	timeField := query.Get("_time_field")
	msgField := query.Get("_msg_field")

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Failed to read body: %v", err))
		atomic.AddInt64(&s.errorCount, 1)
		return
	}
	defer r.Body.Close()

	atomic.AddInt64(&s.totalBytes, int64(len(body)))

	// Parse JSONL format (each line is a JSON object)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	lineCount := 0
	invalidLines := 0

	s.mu.Lock()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lineCount++
		var logData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logData); err != nil {
			if s.config.Verbose {
				log.Printf("Failed to parse log line %d: %v", lineCount, err)
			}
			invalidLines++
			continue
		}

		entry := LogEntry{
			Timestamp:    time.Now(),
			Data:         logData,
			RawData:      line,
			StreamFields: streamFields,
			TimeField:    timeField,
			MsgField:     msgField,
		}
		s.logs = append(s.logs, entry)
	}
	s.mu.Unlock()

	if err := scanner.Err(); err != nil {
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Error reading body: %v", err))
		atomic.AddInt64(&s.errorCount, 1)
		return
	}

	atomic.AddInt64(&s.successCount, 1)

	if s.config.Verbose {
		log.Printf("✅ Received %d logs (invalid: %d) | stream_fields=%v, time_field=%s, msg_field=%s",
			lineCount, invalidLines, streamFields, timeField, msgField)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleHealth handles GET /health
func (s *MockServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&s.healthStatus) == 1 {
		s.sendJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleStats returns server statistics
func (s *MockServer) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	totalLogs := len(s.logs)
	s.mu.RUnlock()

	healthStatus := "healthy"
	if atomic.LoadInt32(&s.healthStatus) == 1 {
		healthStatus = "unhealthy"
	}

	faultMode := "normal"
	switch atomic.LoadInt32(&s.faultMode) {
	case 1:
		faultMode = "error"
	case 2:
		faultMode = "slow"
	}

	stats := Stats{
		TotalRequests: atomic.LoadInt64(&s.requestCount),
		SuccessCount:  atomic.LoadInt64(&s.successCount),
		ErrorCount:    atomic.LoadInt64(&s.errorCount),
		TotalLogs:     totalLogs,
		TotalBytes:    atomic.LoadInt64(&s.totalBytes),
		Uptime:        time.Since(s.startTime).String(),
		StartTime:     s.startTime,
		HealthStatus:  healthStatus,
		FaultMode:     faultMode,
	}

	s.sendJSON(w, http.StatusOK, stats)
}

// handleGetLogs returns stored logs
func (s *MockServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Support pagination
	query := r.URL.Query()
	limit := 100
	if l := query.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Calculate slice bounds
	total := len(s.logs)
	if offset >= total {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"total":  total,
			"offset": offset,
			"limit":  limit,
			"logs":   []LogEntry{},
		})
		return
	}

	end := offset + limit
	if end > total {
		end = total
	}

	result := map[string]interface{}{
		"total":  total,
		"offset": offset,
		"limit":  limit,
		"logs":   s.logs[offset:end],
	}

	s.sendJSON(w, http.StatusOK, result)
}

// handleClear clears all stored logs
func (s *MockServer) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		s.sendError(w, http.StatusMethodNotAllowed, "Only POST or DELETE methods are allowed")
		return
	}

	s.mu.Lock()
	s.logs = make([]LogEntry, 0)
	s.mu.Unlock()

	atomic.StoreInt64(&s.requestCount, 0)
	atomic.StoreInt64(&s.successCount, 0)
	atomic.StoreInt64(&s.errorCount, 0)
	atomic.StoreInt64(&s.totalBytes, 0)

	if s.config.Verbose {
		log.Println("🗑️  All logs and stats cleared")
	}

	s.sendJSON(w, http.StatusOK, map[string]string{
		"message": "All logs and stats cleared",
	})
}

// handleControlHealth controls health status
func (s *MockServer) handleControlHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	status := r.URL.Query().Get("status")
	switch status {
	case "healthy":
		atomic.StoreInt32(&s.healthStatus, 0)
		if s.config.Verbose {
			log.Println("✅ Health status set to: healthy")
		}
	case "unhealthy":
		atomic.StoreInt32(&s.healthStatus, 1)
		if s.config.Verbose {
			log.Println("❌ Health status set to: unhealthy")
		}
	default:
		s.sendError(w, http.StatusBadRequest, "Invalid status. Use 'healthy' or 'unhealthy'")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Health status set to: %s", status),
		"status":  status,
	})
}

// handleControlFault controls fault injection mode
func (s *MockServer) handleControlFault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "normal":
		atomic.StoreInt32(&s.faultMode, 0)
		if s.config.Verbose {
			log.Println("✅ Fault mode set to: normal")
		}
	case "error":
		atomic.StoreInt32(&s.faultMode, 1)
		if s.config.Verbose {
			log.Println("⚠️  Fault mode set to: error")
		}
	case "slow":
		atomic.StoreInt32(&s.faultMode, 2)
		if s.config.Verbose {
			log.Println("🐌 Fault mode set to: slow")
		}
	default:
		s.sendError(w, http.StatusBadRequest, "Invalid mode. Use 'normal', 'error', or 'slow'")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Fault mode set to: %s", mode),
		"mode":    mode,
	})
}

// handleFaultInjection simulates faults based on current mode
func (s *MockServer) handleFaultInjection(w http.ResponseWriter) bool {
	mode := atomic.LoadInt32(&s.faultMode)
	switch mode {
	case 1: // error mode
		s.sendError(w, http.StatusInternalServerError, "Simulated server error")
		return true
	case 2: // slow mode
		time.Sleep(3 * time.Second)
	}
	return false
}

// Helper functions
func (s *MockServer) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *MockServer) sendError(w http.ResponseWriter, status int, message string) {
	errResp := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    status,
	}
	s.sendJSON(w, status, errResp)
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
