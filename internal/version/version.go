package version

import (
	"fmt"
	"runtime"
)

var (
	Version   string // will be populated by linker during build
	BuildDate string // will be populated by linker during build
	Commit    string // will be populated by linker during build
)

// Print returns a human-readable string with build information about the binary.
// Modeled after github.com/prometheus/common/version.Print().
func Print(programName string) string {
	return fmt.Sprintf("%s build info:\n\tversion: %s\n\tbuild date: %s\n\tcommit: %s\n\tgo version: %s\n",
		programName,
		Version,
		BuildDate,
		Commit,
		runtime.Version(),
	)
}

// Info prints build info in a more condensed, single line format.
// Modeled after github.com/prometheus/common/version.Info().
func Info() string {
	return fmt.Sprintf("(version=%s, build_date=%s, commit=%s)", Version, BuildDate, Commit)
}
