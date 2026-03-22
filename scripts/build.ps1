#!/usr/bin/env pwsh
# PowerShell equivalents for all Makefile targets
# Usage: .\scripts\build.ps1 <target>
# Example: .\scripts\build.ps1 build-multiarch-image

param(
    [Parameter(Position = 0)]
    [ValidateSet("build", "run", "lint", "test", "build-image", "build-multiarch-image", "clean", "help")]
    [string]$Target = "help"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Build {
    Write-Host ">> Building Go binary..." -ForegroundColor Cyan
    if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }
    go build -o bin/s3manager.exe .
    Write-Host "Done. Binary at: bin/s3manager.exe" -ForegroundColor Green
}

function Invoke-Run {
    Write-Host ">> Running application..." -ForegroundColor Cyan
    go run .
}

function Invoke-Lint {
    Write-Host ">> Running linter..." -ForegroundColor Cyan
    golangci-lint run
}

function Invoke-Test {
    Write-Host ">> Running tests..." -ForegroundColor Cyan
    go test -race -cover ./...
}

function Invoke-BuildImage {
    Write-Host ">> Building local Docker image..." -ForegroundColor Cyan
    docker build -t s3manager .
    Write-Host "Done. Image tagged as: s3manager" -ForegroundColor Green
}

function Invoke-BuildMultiarchImage {
    Write-Host ">> Setting up Docker Buildx builder..." -ForegroundColor Cyan

    # Try to use existing builder, create it if it doesn't exist
    $builderExists = docker buildx use multiarch-builder 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "   Builder not found, creating multiarch-builder..." -ForegroundColor Yellow
        docker buildx create --name multiarch-builder --use
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to create buildx builder."
            exit 1
        }
    } else {
        Write-Host "   Using existing multiarch-builder." -ForegroundColor Gray
    }

    Write-Host ">> Building and pushing multi-arch image (linux/amd64 + linux/arm64)..." -ForegroundColor Cyan
    Write-Host "   Tags: dimuthnc/s3manager:latest, dimuthnc/s3manager:v1.2.0" -ForegroundColor Gray
    docker buildx build `
        --platform linux/amd64,linux/arm64 `
        -t dimuthnc/s3manager:latest `
        -t dimuthnc/s3manager:v1.2.0 `
        --push .

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Done. Pushed to Docker Hub successfully." -ForegroundColor Green
    } else {
        Write-Error "Build/push failed."
        exit 1
    }
}

function Invoke-Clean {
    Write-Host ">> Cleaning build artifacts..." -ForegroundColor Cyan
    if (Test-Path "bin") {
        Remove-Item -Recurse -Force "bin"
        Write-Host "Removed: bin/" -ForegroundColor Green
    } else {
        Write-Host "Nothing to clean." -ForegroundColor Gray
    }
}

function Show-Help {
    Write-Host ""
    Write-Host "Usage: .\scripts\build.ps1 <target>" -ForegroundColor White
    Write-Host ""
    Write-Host "Available targets:" -ForegroundColor Yellow
    Write-Host "  build                 Build the Go binary to bin/s3manager.exe"
    Write-Host "  run                   Run the application locally"
    Write-Host "  lint                  Run golangci-lint"
    Write-Host "  test                  Run all tests with race detection and coverage"
    Write-Host "  build-image           Build a local Docker image tagged as 's3manager'"
    Write-Host "  build-multiarch-image Build and push amd64+arm64 image to Docker Hub"
    Write-Host "  clean                 Remove the bin/ directory"
    Write-Host "  help                  Show this help message"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  .\scripts\build.ps1 build-multiarch-image"
    Write-Host "  .\scripts\build.ps1 test"
    Write-Host "  .\scripts\build.ps1 build"
    Write-Host ""
}

switch ($Target) {
    "build"                 { Invoke-Build }
    "run"                   { Invoke-Run }
    "lint"                  { Invoke-Lint }
    "test"                  { Invoke-Test }
    "build-image"           { Invoke-BuildImage }
    "build-multiarch-image" { Invoke-BuildMultiarchImage }
    "clean"                 { Invoke-Clean }
    "help"                  { Show-Help }
}

