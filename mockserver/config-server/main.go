package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	port       = flag.Int("port", 8080, "HTTP server port")
	configFile = flag.String("config", "configs/vmlogs-test-tasks.json", "Path to configuration JSON file")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	hotReload  = flag.Bool("hot-reload", true, "Enable hot reload of config file")
)

type ConfigServer struct {
	mu         sync.RWMutex
	configData map[string]interface{}
	configFile string
	verbose    bool
	lastMod    time.Time
}

func NewConfigServer(configFile string, verbose bool) (*ConfigServer, error) {
	server := &ConfigServer{
		configFile: configFile,
		verbose:    verbose,
	}

	if err := server.loadConfig(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *ConfigServer) loadConfig() error {
	data, err := ioutil.ReadFile(s.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	s.mu.Lock()
	s.configData = config
	s.lastMod = time.Now()
	s.mu.Unlock()

	if s.verbose {
		log.Printf("✅ Config loaded from %s", s.configFile)
	}

	return nil
}

func (s *ConfigServer) reloadConfigIfNeeded() error {
	fileInfo, err := os.Stat(s.configFile)
	if err != nil {
		return err
	}

	s.mu.RLock()
	lastMod := s.lastMod
	s.mu.RUnlock()

	if fileInfo.ModTime().After(lastMod) {
		if s.verbose {
			log.Printf("🔄 Config file changed, reloading...")
		}
		return s.loadConfig()
	}

	return nil
}

// handleInsightAPI 模拟 InsightService API
// GET /api/v2/dimensions/logevent/tasks
func (s *ConfigServer) handleInsightAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	data := s.configData["data"]
	s.mu.RUnlock()

	if s.verbose {
		log.Printf("📥 InsightService API called from %s", r.RemoteAddr)
	}

	// InsightService 返回格式：不包含 label_mappings
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"data_sources": getField(data, "data_sources"),
			"tasks":        getField(data, "tasks"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleN9eAPI 模拟 N9eService API
// GET /v1/n9e-plus/logevent/tasks
func (s *ConfigServer) handleN9eAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查 Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="N9e Mock"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 这里可以配置期望的用户名密码，或者不验证
	if s.verbose {
		log.Printf("📥 N9eService API called from %s (user: %s)", r.RemoteAddr, username)
	}

	s.mu.RLock()
	data := s.configData
	s.mu.RUnlock()

	// N9eService 返回完整格式：包含 label_mappings
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleStatus 返回服务器状态
func (s *ConfigServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"status":      "running",
		"config_file": s.configFile,
		"last_loaded": s.lastMod.Format(time.RFC3339),
		"endpoints": map[string]string{
			"insight_api": "/api/v2/dimensions/logevent/tasks",
			"n9e_api":     "/v1/n9e-plus/logevent/tasks",
			"status":      "/status",
			"reload":      "/reload (POST)",
		},
	}

	data := s.configData["data"]
	if dataMap, ok := data.(map[string]interface{}); ok {
		if dataSources, ok := dataMap["data_sources"].(map[string]interface{}); ok {
			status["data_sources_count"] = len(dataSources)
		}
		if tasks, ok := dataMap["tasks"].([]interface{}); ok {
			status["tasks_count"] = len(tasks)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleReload 手动触发配置重载
func (s *ConfigServer) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.loadConfig(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reload config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Config reloaded successfully",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func getField(data interface{}, field string) interface{} {
	if dataMap, ok := data.(map[string]interface{}); ok {
		return dataMap[field]
	}
	return nil
}

func main() {
	flag.Parse()

	// 检查配置文件
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		log.Fatalf("❌ Config file not found: %s", *configFile)
	}

	// 创建配置服务器
	server, err := NewConfigServer(*configFile, *verbose)
	if err != nil {
		log.Fatalf("❌ Failed to create config server: %v", err)
	}

	// 设置 HTTP 路由
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/dimensions/logevent/tasks", server.handleInsightAPI)
	mux.HandleFunc("/v1/n9e-plus/logevent/tasks", server.handleN9eAPI)
	mux.HandleFunc("/status", server.handleStatus)
	mux.HandleFunc("/reload", server.handleReload)

	// 启动热重载监控
	if *hotReload {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := server.reloadConfigIfNeeded(); err != nil && *verbose {
					log.Printf("⚠️  Failed to reload config: %v", err)
				}
			}
		}()
	}

	// 启动 HTTP 服务器
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	go func() {
		log.Printf("🚀 Mock Config Server started on http://localhost:%d", *port)
		log.Printf("📝 Config file: %s", *configFile)
		log.Printf("📋 Endpoints:")
		log.Printf("   - InsightService: http://localhost:%d/api/v2/dimensions/logevent/tasks", *port)
		log.Printf("   - N9eService:     http://localhost:%d/v1/n9e-plus/logevent/tasks", *port)
		log.Printf("   - Status:         http://localhost:%d/status", *port)
		log.Printf("   - Reload:         http://localhost:%d/reload (POST)", *port)
		if *hotReload {
			log.Printf("🔄 Hot reload: Enabled")
		}

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")
	if err := httpServer.Close(); err != nil {
		log.Printf("❌ Server shutdown error: %v", err)
	}
	log.Println("✅ Server stopped")
}
