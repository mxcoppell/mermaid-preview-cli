package version

// Version is set at build time via ldflags:
//
//	go build -ldflags="-X github.com/mxcoppell/mermaid-preview-cli/internal/version.Version=1.0.0"
var Version = "dev"
