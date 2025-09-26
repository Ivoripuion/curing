# Changelog

All notable changes to this enhanced version of Curing will be documented in this file.

## [Enhanced v1.0.0] - 2025-09-26

### 🆕 Added
- **Interactive file reading mode** - Real-time file access with command-line interface
- **Flexible networking support** - Custom IP addresses and ports for both server and client
- **Command-line argument system** - Override config files with CLI parameters
- **Built-in help system** - `--help` flag for both server and client
- **Enhanced error handling** - Better error messages and connection failure handling
- **Configuration validation** - Port range and IP address validation
- **Comprehensive documentation** - Quick start guide and usage examples

### 🔧 Enhanced
- **Server networking** - Support for binding to specific interfaces (0.0.0.0, localhost, etc.)
- **Client connectivity** - Connect to any remote server with custom IP/port
- **Configuration system** - Command-line parameters override config file settings
- **Deployment flexibility** - Easy setup for local testing and remote operations

### 🛠️ Technical Improvements
- **Server architecture** - Enhanced to handle `ReadFileRequest` type
- **Client modes** - Support for both interactive and traditional C2 modes
- **Request/response protocol** - Extended with new request types
- **io_uring integration** - All file and network operations use io_uring (no traditional syscalls)

### 📚 Documentation
- **Reorganized README** - Clear distinction between original and enhanced features
- **Quick Start Guide** - 5-minute setup and usage guide
- **Usage examples** - Local and remote deployment scenarios
- **Troubleshooting guide** - Common issues and solutions

## [Original] - Base Version

### Original Features (by amitschendel)
- Basic C2 communication using io_uring
- File read/write operations via io_uring
- Symbolic link creation via io_uring
- Syscall bypass demonstration
- Falco security tool bypass POC

---

**Note**: This enhanced version builds upon the original Curing project by [amitschendel](https://github.com/amitschendel/curing), adding interactive capabilities and deployment flexibility while maintaining the core io_uring bypass techniques.
