package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/amitschendel/curing/pkg/config"
	"github.com/amitschendel/curing/pkg/server"
)

func main() {
	// Command line flags
	port := flag.Int("port", 0, "Server port (overrides config file)")
	host := flag.String("host", "", "Server host/bind address (overrides config file)")
	configFile := flag.String("config", "cmd/config.json", "Path to config file")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Curing Server - io_uring based C2 server")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  ./server                           # Use default config")
		fmt.Println("  ./server -port 9999               # Override port")
		fmt.Println("  ./server -host 0.0.0.0 -port 8080 # Bind to all interfaces")
		fmt.Println("  ./server -config /path/to/config  # Use custom config file")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Printf("Warning: Could not load config file (%v), using defaults", err)
		cfg = &config.Config{
			Server: config.ServerDetails{
				Host: "localhost",
				Port: 8888,
			},
		}
	}

	// Override config with command line arguments
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}

	// Validate configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		log.Fatalf("Invalid port: %d (must be 1-65535)", cfg.Server.Port)
	}

	fmt.Printf("Starting Curing Server on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	s := server.NewServer(cfg.Server.Host, cfg.Server.Port)
	s.Run()
}
