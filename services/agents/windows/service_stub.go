//go:build !windows

package windows

import "fmt"

// ConfigPath matches the Windows build and is provided for callers that may
// reference it when cross-compiling.
const ConfigPath = `C:\\ProgramData\\Goosed\\agent.conf`

// Run is a stub used when building the agent on non-Windows platforms.
func Run() error {
	return fmt.Errorf("goosed windows agent is only supported on Windows")
}
