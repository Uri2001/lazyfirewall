//go:build linux
// +build linux

package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("LazyFirewall %s (%s, %s)", Version, Commit, Date)
}
