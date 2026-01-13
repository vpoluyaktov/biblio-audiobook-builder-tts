package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

// KillProcessOnPort kills the process listening on the specified port
// Returns nil if a process was killed, an error if something went wrong,
// or a special error if no process was found
func KillProcessOnPort(port string) error {
	var checkCmd, killCmd *exec.Cmd

	switch runtime.GOOS {
	case "linux", "darwin":
		// Find the process LISTENING on the port (not just connected to it)
		// Use -sTCP:LISTEN to only match listening sockets
		checkCmd = exec.Command("sh", "-c", fmt.Sprintf("lsof -ti:%s -sTCP:LISTEN", port))
		output, err := checkCmd.CombinedOutput()

		// If lsof returns nothing or errors, no process is listening on the port
		if err != nil || len(output) == 0 {
			return fmt.Errorf("no process found on port %s", port)
		}

		// Kill only the process that is LISTENING on the port
		killCmd = exec.Command("sh", "-c",
			fmt.Sprintf("lsof -ti:%s -sTCP:LISTEN | xargs kill -9 2>/dev/null", port))
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("failed to kill process on port %s", port)
		}

	case "windows":
		// Check if there's a process on the port
		checkCmd = exec.Command("cmd", "/C",
			fmt.Sprintf("netstat -ano | findstr :%s", port))
		output, err := checkCmd.CombinedOutput()

		if err != nil || len(output) == 0 {
			return fmt.Errorf("no process found on port %s", port)
		}

		// Kill the process
		killCmd = exec.Command("cmd", "/C",
			fmt.Sprintf("for /f \"tokens=5\" %%%%a in ('netstat -ano ^| findstr :%s') do taskkill /F /PID %%%%a", port))
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("failed to kill process on port %s", port)
		}

	default:
		return fmt.Errorf("unsupported platform for restart")
	}

	return nil
}
