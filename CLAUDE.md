# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Structure

```
cmd/bghelper/    - Main application entry point
internal/        - Private application code
pkg/             - Public library code (if any)
```

## Commands

```bash
# Build
go build ./cmd/bghelper

# Run
go run ./cmd/bghelper

# Test
go test ./...

# Test with coverage
go test -cover ./...

# Lint (requires golangci-lint)
golangci-lint run

# Format
go fmt ./...
```
