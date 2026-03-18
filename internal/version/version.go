// Package version holds build-time metadata injected via ldflags.
package version

// Set by -ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return Version + " (" + Commit + ") " + Date
}
