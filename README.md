# Registry Mirror 🐳

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](.)
[![Coverage](https://img.shields.io/badge/coverage-85%25-brightgreen.svg)](.)

A smart CLI tool to mirror and cache Docker images locally with advanced features including interactive TUI, time-based policies, Git integration, and signature verification.

Born from the frustration of slow image pulls on home networks, this tool makes your container workflow lightning fast by intelligently caching popular images.

## 🚀 Why I Built This

I often work with large ML images (TensorFlow, PyTorch) and microservices on my home lab. Pulling `tensorflow/tensorflow:latest` (approx 1GB+) would take minutes every time I reset my environment.

I built `registry-mirror` to:
- **Slash pull times**: From ~5 mins to <5 seconds for cached images
- **Save Bandwidth**: Why download the same Alpine layer 50 times?
- **Work Offline**: Keep developing even when the internet drops

## ✨ Features

- **Interactive TUI Dashboard**: Beautiful terminal interface with real-time monitoring and keyboard navigation
- **Time-Based Policies**: TTL-based cache expiration with temporary access grants and audit logging
- **Git Integration**: Automatically scan repositories for container images in Dockerfiles, K8s manifests, and Docker Compose files
- **Signature Verification**: Cryptographic verification using Cosign/Sigstore for supply chain security
- **Comprehensive Testing**: 85%+ test coverage with CI/CD pipeline
- **Smart Sync**: Parallel layer downloading for maximum speed
- **Analytics Dashboard**: See exactly how much time and bandwidth you've saved
- **Auto-Mirror**: Predicts and pre-fetches popular images (Node, Postgres, etc.)
- **Cache Policy**: LRU eviction to keep your disk usage under control
- **Health Checks**: Built-in diagnostics for your registry setup
- **Audit Logging**: Complete trail of all policy changes for compliance

## 📦 Installation

```bash
git clone https://github.com/yourusername/registry-mirror
cd registry-mirror
go install
```

## 🛠️ Usage

### Prerequisite
You need a local registry running (standard Docker registry):
```bash
docker run -d -p 5000:5000 --restart=always --name registry registry:2
```

### 0. Launch Interactive TUI (Recommended)
Experience the full power with the interactive dashboard:
```bash
registry-mirror tui
```

Features:
- Real-time monitoring of sync operations
- Navigate cached images with keyboard shortcuts (j/k or arrow keys)
- View detailed statistics and bandwidth savings
- Multiple views: Dashboard, Images, Stats, Logs
- Press `Tab` to switch views, `r` to refresh, `q` to quit

### 1. Sync an Image
Mirror an image to your local registry:
```bash
registry-mirror sync nginx:latest
```

### 2. Check Status
See what's in your mirror:
```bash
registry-mirror status
```

### 3. View Analytics
See your savings:
```bash
registry-mirror analytics
```

### 4. Auto-Mirror
Let the tool find popular images you might need:
```bash
registry-mirror auto --top 10
```

### 5. Time-Based Policies
Manage cache with TTL-based expiration:
```bash
# Set a 24-hour cache policy
registry-mirror policy set nginx:latest --ttl 24h --reason "Daily refresh"

# Grant temporary 2-hour access
registry-mirror allow redis:latest --expires-in 2h --reason "Quick test"

# List active policies
registry-mirror policy list

# View audit log
registry-mirror policy audit

# Clean up expired policies
registry-mirror policy cleanup
```

### 6. Git Repository Scanning
Scan repositories for container images:
```bash
# Scan current directory
registry-mirror scan-repo

# Scan specific path
registry-mirror scan-repo /path/to/repo

# Generate Bill of Materials
registry-mirror scan-repo --bom images-bom.md
```

### 7. Signature Verification
Verify image signatures for security:
```bash
# Verify single image
registry-mirror verify nginx:latest

# Verify with requirement
registry-mirror verify gcr.io/distroless/static:latest --require-signature

# Sync with verification
registry-mirror sync nginx:latest --verify-signature
```

## 📚 Documentation

- **[QUICK-START.md](QUICK-START.md)** - Quick reference for all commands
- **[FEATURES.md](FEATURES.md)** - Detailed feature implementation plan  
- **[TESTING.md](TESTING.md)** - Testing guide and best practices
- **[TEST-RESULTS.md](TEST-RESULTS.md)** - Actual test results
- **[test-features.md](test-features.md)** - Step-by-step testing guide

## ⚙️ Configuration

Create a `.registry-mirror.yaml` in your home directory:

```yaml
registry: "localhost:5000"
parallel: 5
cache_limit_mb: 20000
```

## 📈 Performance

| Image | Docker Hub Pull | Local Mirror Pull |
|-------|-----------------|-------------------|
| Nginx | ~15s | **~2s** |
| Postgres | ~45s | **~5s** |
| TensorFlow | ~4m 30s | **~15s** |

## 🤝 Contributing

This started as a weekend project but I'm open to PRs! Please keep code simple and readable.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/saurabh12nxf/registry-mirror
cd registry-mirror

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o registry-mirror.exe
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run benchmarks
make bench
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Styling with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Security with [Cosign](https://github.com/sigstore/cosign)

## 📧 Contact

- GitHub: [@saurabh12nxf](https://github.com/saurabh12nxf)
- Project: [registry-mirror](https://github.com/saurabh12nxf/registry-mirror)

---

**Built with ❤️ for the container community**
