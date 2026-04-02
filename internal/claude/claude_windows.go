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

// DataDir 返回 Claude 应用数据目录
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

// IsRunning 检查 Claude 是否正在运行
func IsRunning() bool {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+exeName, "/NH").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), exeName)
}

// Quit 关闭 Claude 应用
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

	// 超时后强制终止
	_ = exec.Command("taskkill", "/F", "/IM", exeName).Run()
	time.Sleep(500 * time.Millisecond)

	if IsRunning() {
		return fmt.Errorf("Claude failed to close within timeout")
	}

	return nil
}

// findExe 查找 Claude 可执行文件路径
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

	// 回退到 PATH 查找
	p, err := exec.LookPath(exeName)
	if err == nil {
		return p, nil
	}

	return "", fmt.Errorf("Claude executable not found")
}

// Launch 启动 Claude 应用
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

// LaunchWithDataDir 以独立数据目录启动新的 Claude 实例
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

// IsInstanceRunning 检查指定数据目录的 Claude 实例是否在运行
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

// CloseInstance 关闭指定数据目录的 Claude 实例
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
