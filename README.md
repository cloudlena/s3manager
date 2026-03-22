# S3 Manager

A Web GUI written in Go to manage S3 buckets from any provider.

> **Note:** This is a fork of [cloudlena/s3manager](https://github.com/cloudlena/s3manager) with a completely redesigned UI using [shadcn/ui](https://ui.shadcn.com/) design principles.

## What's Different in This Fork?

- 🎨 **Modern UI** - Completely revamped interface using shadcn/ui design system
- 🧹 **Clean Design** - Replaced Materialize CSS with custom shadcn/ui-based styling
- 📦 **Lightweight** - SVG icons instead of Material Icons font dependency
- ✨ **Better UX** - Improved modals, toasts, tables, and empty states

## Docker Image

This custom image is available on Docker Hub:

```bash
# Latest version
docker pull dimuthnc/s3manager:latest

# Specific version
docker pull dimuthnc/s3manager:v1.2.0
```

**Available Tags:**
| Tag | Description |
|-----|-------------|
| `dimuthnc/s3manager:latest` | Latest build |
| `dimuthnc/s3manager:v1.2.0` | Latest release with UI improvements and sample data |
| `dimuthnc/s3manager:v1.0.0` | Initial release with shadcn/ui design |

## Features

- List all buckets in your account
- Create a new bucket
- List all objects in a bucket
- Upload new objects to a bucket
- Download objects from a bucket
- Delete objects in a bucket
- Generate presigned download URLs
- Switch between multiple S3 instances

## Quick Start

```bash
docker run -p 8080:8080 \
  -e 'ACCESS_KEY_ID=your-access-key' \
  -e 'SECRET_ACCESS_KEY=your-secret-key' \
  -e 'ENDPOINT=s3.amazonaws.com' \
  dimuthnc/s3manager:v1.2.0
```

Then visit <http://localhost:8080>

## Configuration

The application can be configured with the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENDPOINT` | The endpoint of your S3 server | `s3.amazonaws.com` |
| `REGION` | The region of your S3 server | `""` |
| `ACCESS_KEY_ID` | Your S3 access key ID (required if `USE_IAM` is `false`) | - |
| `SECRET_ACCESS_KEY` | Your S3 secret access key (required if `USE_IAM` is `false`) | - |
| `USE_SSL` | Whether your S3 server uses SSL | `true` |
| `SKIP_SSL_VERIFICATION` | Skip SSL verification | `false` |
| `SIGNATURE_TYPE` | Signature type (`V2`, `V4`, `V4Streaming`, `Anonymous`) | `V4` |
| `PORT` | Port the app listens on | `8080` |
| `ALLOW_DELETE` | Enable delete buttons | `true` |
| `FORCE_DOWNLOAD` | Force download instead of opening in browser | `true` |
| `LIST_RECURSIVE` | List all objects recursively | `false` |
| `USE_IAM` | Use IAM role instead of key pair | `false` |
| `IAM_ENDPOINT` | Endpoint for IAM role retrieving | `""` |
| `SSE_TYPE` | Server side encryption (`SSE`, `KMS`, `SSE-C`) | `""` |
| `SSE_KEY` | Key for SSE (only for `KMS` and `SSE-C`) | `""` |
| `TIMEOUT` | Read and write timeout in seconds | `600` |
| `ROOT_URL` | Root URL prefix for reverse proxy | `""` |

## Usage Examples

### With MinIO

```bash
docker run -p 8080:8080 \
  -e 'ENDPOINT=play.min.io' \
  -e 'ACCESS_KEY_ID=minioadmin' \
  -e 'SECRET_ACCESS_KEY=minioadmin' \
  dimuthnc/s3manager:v1.2.0
```

### With AWS S3

```bash
docker run -p 8080:8080 \
  -e 'ENDPOINT=s3.amazonaws.com' \
  -e 'REGION=us-east-1' \
  -e 'ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE' \
  -e 'SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY' \
  dimuthnc/s3manager:v1.2.0
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  s3manager:
    image: dimuthnc/s3manager:v1.2.0
    ports:
      - "8080:8080"
    environment:
      - ENDPOINT=s3.amazonaws.com
      - ACCESS_KEY_ID=your-access-key
      - SECRET_ACCESS_KEY=your-secret-key
```

### Deploy to Kubernetes

You can adapt the [Helm chart](https://github.com/sergeyshevch/s3manager-helm) for use with this image by updating the image repository to `dimuthnc/s3manager`.

## Development

Two ways to run development commands are available depending on your OS:
- **Linux / macOS** — use `make`
- **Windows** — use the PowerShell script at `scripts/build.ps1`

### Build and Run Locally

```bash
# Linux / macOS
make build
./bin/s3manager
```

```powershell
# Windows
.\scripts\build.ps1 build
.\bin\s3manager.exe
```

### Run Tests

```bash
# Linux / macOS
make test
```

```powershell
# Windows
.\scripts\build.ps1 test
```

### Lint Code

```bash
# Linux / macOS
make lint
```

```powershell
# Windows
.\scripts\build.ps1 lint
```

### Build Container Image

```bash
# Linux / macOS
make build-image
```

```powershell
# Windows
.\scripts\build.ps1 build-image
```

### Build & Push Multi-Architecture Image (amd64 + arm64)

```bash
# Linux / macOS — login to Docker Hub first
docker login

make build-multiarch-image
```

```powershell
# Windows — login to Docker Hub first
docker login

.\scripts\build.ps1 build-multiarch-image
```

This creates a single image that works on both **x86_64** (Intel/AMD) and **arm64** (Apple Silicon, AWS Graviton) machines and pushes two tags: `latest` and `v1.2.0`.

### Clean Build Artifacts

```bash
# Linux / macOS
make clean
```

```powershell
# Windows
.\scripts\build.ps1 clean
```

### All Available PowerShell Script Commands

| Command | Description |
|---|---|
| `.\scripts\build.ps1 build` | Build the Go binary to `bin/s3manager.exe` |
| `.\scripts\build.ps1 run` | Run the application locally |
| `.\scripts\build.ps1 test` | Run all tests with race detection and coverage |
| `.\scripts\build.ps1 lint` | Run golangci-lint |
| `.\scripts\build.ps1 build-image` | Build a local Docker image tagged as `s3manager` |
| `.\scripts\build.ps1 build-multiarch-image` | Build and push amd64+arm64 image to Docker Hub |
| `.\scripts\build.ps1 clean` | Remove the `bin/` directory |
| `.\scripts\build.ps1 help` | Show all available commands |

> **Note (Windows):** If you get a script execution policy error, run the following once in an elevated PowerShell session:
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
> ```

### Run with Docker Compose (Local Development)

```bash
docker-compose up
```

## Credits

This project is a fork of [cloudlena/s3manager](https://github.com/cloudlena/s3manager) by Lena Fuhrimann.

**Modifications in this fork:**
- Completely revamped UI using [shadcn/ui](https://ui.shadcn.com/) design system
- Replaced Materialize CSS with custom CSS based on shadcn/ui components
- Modern, clean interface with improved user experience
- SVG icons replacing Material Icons font dependency

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Original work Copyright 2016 Lena Fuhrimann.
