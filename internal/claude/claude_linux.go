package claude

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const appName = "claude"

// DataDir returns the Claude application data directory
func DataDir() (string, error) {
	// Prefer XDG_CONFIG_HOME if set
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "Claude"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}

	return filepath.Join(home, ".config", "Claude"), nil
}

// IsRunning checks whether Claude is currently running
func IsRunning() bool {
	out, err := exec.Command("pgrep", "-x", appName).Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) != ""
}

// Quit gracefully closes the Claude app
func Quit() error {
	if !IsRunning() {
		return nil
	}

	_ = exec.Command("pkill", "-x", appName).Run()

	for range 30 {
		if !IsRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Force kill on timeout
	_ = exec.Command("pkill", "-9", "-x", appName).Run()
	time.Sleep(500 * time.Millisecond)

	if IsRunning() {
		return fmt.Errorf("Claude failed to close within timeout")
	}

	return nil
}

// Launch starts the Claude app
func Launch() error {
	cmd := exec.Command(appName)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// LaunchWithDataDir launches a new Claude instance with an isolated data directory
func LaunchWithDataDir(dataDir string) error {
	cmd := exec.Command(appName, "--user-data-dir="+dataDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// IsInstanceRunning checks whether a Claude instance with the given data directory is running
func IsInstanceRunning(dataDir string) bool {
	out, err := exec.Command("pgrep", "-f", "user-data-dir="+dataDir).Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) != ""
}

// CloseInstance closes the Claude instance with the given data directory
func CloseInstance(dataDir string) error {
	if !IsInstanceRunning(dataDir) {
		return nil
	}

	killInstanceProcs(dataDir, false)

	for range 30 {
		if !IsInstanceRunning(dataDir) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	killInstanceProcs(dataDir, true)
	time.Sleep(500 * time.Millisecond)

	if IsInstanceRunning(dataDir) {
		return fmt.Errorf("instance failed to close within timeout")
	}

	return nil
}

func killInstanceProcs(dataDir string, force bool) {
	out, err := exec.Command("pgrep", "-f", "user-data-dir="+dataDir).Output()
	if err != nil {
		return
	}

	sig := "TERM"
	if force {
		sig = "KILL"
	}

	for _, pid := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-s", sig, pid).Run()
	}
}
