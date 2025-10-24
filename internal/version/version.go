package version

import "fmt"

var (
	// Version is the semantic version (injected via ldflags at build time)
	Version = "dev"

	// GitCommit is the git commit hash (injected via ldflags)
	GitCommit = "none"

	// BuildDate is the build timestamp (injected via ldflags)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build (injected via ldflags)
	GoVersion = "unknown"
)

// Info returns structured version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// Get returns the version information as a struct
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}

// String returns a human-readable version string
func String() string {
	return fmt.Sprintf("twigbush %s", Version)
}

// Verbose returns a detailed version string
func Verbose() string {
	return fmt.Sprintf("twigbush %s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, GoVersion)
}
