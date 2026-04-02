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

// DataDir 返回 Claude 应用数据目录
func DataDir() (string, error) {
	// 优先使用 XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "Claude"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}

	return filepath.Join(home, ".config", "Claude"), nil
}

// IsRunning 检查 Claude 是否正在运行
func IsRunning() bool {
	out, err := exec.Command("pgrep", "-x", appName).Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) != ""
}

// Quit 关闭 Claude 应用
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

	// 超时后强制终止
	_ = exec.Command("pkill", "-9", "-x", appName).Run()
	time.Sleep(500 * time.Millisecond)

	if IsRunning() {
		return fmt.Errorf("Claude failed to close within timeout")
	}

	return nil
}

// Launch 启动 Claude 应用
func Launch() error {
	cmd := exec.Command(appName)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// LaunchWithDataDir 以独立数据目录启动新的 Claude 实例
func LaunchWithDataDir(dataDir string) error {
	cmd := exec.Command(appName, "--user-data-dir="+dataDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// IsInstanceRunning 检查指定数据目录的 Claude 实例是否在运行
func IsInstanceRunning(dataDir string) bool {
	out, err := exec.Command("pgrep", "-f", "user-data-dir="+dataDir).Output()
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
