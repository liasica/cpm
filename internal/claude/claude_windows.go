package claude

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	appName = "Claude"
	exeName = "Claude.exe"
)

// DataDir returns the Claude application data directory
func DataDir() (string, error) {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Claude"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}

	return filepath.Join(home, "AppData", "Roaming", "Claude"), nil
}

// IsRunning checks whether Claude is currently running
func IsRunning() bool {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+exeName, "/NH").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), exeName)
}

// Quit gracefully closes the Claude app
func Quit() error {
	if !IsRunning() {
		return nil
	}

	_ = exec.Command("taskkill", "/IM", exeName).Run()

	for range 30 {
		if !IsRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Force kill on timeout
	_ = exec.Command("taskkill", "/F", "/IM", exeName).Run()
	time.Sleep(500 * time.Millisecond)

	if IsRunning() {
		return fmt.Errorf("Claude failed to close within timeout")
	}

	return nil
}

// findExe locates the Claude executable path
func findExe() (string, error) {
	candidates := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "claude-desktop", exeName),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Claude", exeName),
		filepath.Join(os.Getenv("LOCALAPPDATA"), appName, exeName),
		filepath.Join(os.Getenv("PROGRAMFILES"), appName, exeName),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Fall back to PATH lookup
	p, err := exec.LookPath(exeName)
	if err == nil {
		return p, nil
	}

	return "", fmt.Errorf("Claude executable not found")
}

// Launch starts the Claude app
func Launch() error {
	exePath, err := findExe()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// LaunchWithDataDir launches a new Claude instance with an isolated data directory
func LaunchWithDataDir(dataDir string) error {
	exePath, err := findExe()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, "--user-data-dir="+dataDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// IsInstanceRunning checks whether a Claude instance with the given data directory is running
func IsInstanceRunning(dataDir string) bool {
	script := fmt.Sprintf(
		`Get-CimInstance Win32_Process -Filter "Name='%s'" | Where-Object { $_.CommandLine -like '*user-data-dir=%s*' } | Select-Object -First 1 ProcessId`,
		exeName, dataDir,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
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
	script := fmt.Sprintf(
		`Get-CimInstance Win32_Process -Filter "Name='%s'" | Where-Object { $_.CommandLine -like '*user-data-dir=%s*' } | ForEach-Object { $_.ProcessId }`,
		exeName, dataDir,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return
	}

	for _, pid := range strings.Fields(strings.TrimSpace(string(out))) {
		if force {
			_ = exec.Command("taskkill", "/F", "/PID", pid).Run()
		} else {
			_ = exec.Command("taskkill", "/PID", pid).Run()
		}
	}
}
