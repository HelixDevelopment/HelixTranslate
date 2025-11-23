# AGENTS.md - Universal Multi-Format Multi-Language Ebook Translation System

## Build/Lint/Test Commands
- **Build**: `make build` or `go build ./cmd/cli`
- **Test all**: `make test-unit test-integration test-e2e`
- **Test single**: `go test -v -run TestFunctionName ./pkg/package`
- **Lint**: `make lint` (golangci-lint)
- **Format**: `make fmt` (go fmt)

## Code Style Guidelines
- **Go version**: 1.25.2, **Module**: `digital.vasic.translator`
- **Naming**: PascalCase for exported, camelCase for unexported
- **Imports**: Standard library → third-party → local packages (alphabetical within groups)
- **Types**: Use `any` instead of `interface{}`, interfaces for behavior, composition over inheritance
- **Error handling**: Explicit returns, wrap with context: `fmt.Errorf("failed: %w", err)`
- **Testing**: Table-driven tests, naming: `TestFunctionName_Scenario`
- **Comments**: Document exported functions/types, avoid obvious comments
- **Security**: Never hardcode API keys, use environment variables