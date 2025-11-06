package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	port        = flag.Int("port", 9428, "HTTP server port")
	enableAuth  = flag.Bool("auth", false, "Enable basic authentication")
	username    = flag.String("username", "root", "Basic auth username")
	password    = flag.String("password", "root.2020", "Basic auth password")
	enableFault = flag.Bool("fault", false, "Enable fault injection mode")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	config := &ServerConfig{
		Port:        *port,
		EnableAuth:  *enableAuth,
		Username:    *username,
		Password:    *password,
		EnableFault: *enableFault,
		Verbose:     *verbose,
	}

	server := NewMockServer(config)

	// Start server
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	fmt.Printf("🚀 VictoriaLogs Mock Server started on http://localhost:%d\n", *port)
	fmt.Printf("📝 Insert endpoint: POST http://localhost:%d/insert/jsonline\n", *port)
	fmt.Printf("❤️  Health check: GET http://localhost:%d/health\n", *port)
	fmt.Printf("📊 Stats endpoint: GET http://localhost:%d/stats\n", *port)
	fmt.Printf("🔍 Query logs: GET http://localhost:%d/logs\n", *port)
	if *enableAuth {
		fmt.Printf("🔐 Authentication: Enabled (user: %s)\n", *username)
	}
	if *enableFault {
		fmt.Printf("⚠️  Fault injection: Enabled\n")
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")
	server.Stop()
	fmt.Println("✅ Server stopped")
}
