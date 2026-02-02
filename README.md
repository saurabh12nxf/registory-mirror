# Registry Mirror 🐳

A smart CLI tool to mirror and cache Docker images locally. Born from the frustration of slow image pulls on home networks, this tool makes your container workflow lightning fast by intelligently caching popular images.

## 🚀 Why I Built This

I often work with large ML images (TensorFlow, PyTorch) and microservices on my home lab. Pulling `tensorflow/tensorflow:latest` (approx 1GB+) would take minutes every time I reset my environment.

I built `registry-mirror` to:
- **Slash pull times**: From ~5 mins to <5 seconds for cached images
- **Save Bandwidth**: Why download the same Alpine layer 50 times?
- **Work Offline**: Keep developing even when the internet drops

## ✨ Features

- **Smart Sync**: Parallel layer downloading for maximum speed
- **Analytics Dashboard**: See exactly how much time and bandwidth you've saved
- **Auto-Mirror**: Predicts and pre-fetches popular images (Node, Postgres, etc.)
- **Cache Policy**: LRU eviction to keep your disk usage under control
- **Health Checks**: Built-in diagnostics for your registry setup

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

## License

MIT
