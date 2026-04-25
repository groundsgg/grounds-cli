package version

// These are populated via -ldflags by the Makefile and goreleaser:
//   -X github.com/groundsgg/grounds-cli/internal/version.Version=...
var (
	Version = "dev"
	Commit  = "none"
	BuildAt = "unknown"
)
