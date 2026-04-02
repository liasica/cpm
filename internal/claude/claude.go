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
	appName  = "Claude"
	bundleID = "com.anthropic.claudefordesktop"
)

// 需要按 profile 切换的认证相关条目
var authEntries = []string{
	"Cookies",
	"Cookies-journal",
	"Local Storage",
	"Session Storage",
}

// 需要同步到独立实例的共享配置
var sharedConfigs = []string{
	"claude_desktop_config.json",
	"config.json",
}

// DataDir 返回 Claude 应用数据目录
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	return filepath.Join(home, "Library", "Application Support", "Claude"), nil
}

// AuthEntries 返回需要切换的认证条目列表
func AuthEntries() []string {
	return authEntries
}

// SharedConfigs 返回需要同步到独立实例的共享配置列表
func SharedConfigs() []string {
	return sharedConfigs
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

	// 优雅退出
	_ = exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to quit`, appName)).Run()

	// 等待进程退出
	for range 30 {
		if !IsRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// 超时后强制终止
	_ = exec.Command("pkill", "-x", appName).Run()
	time.Sleep(500 * time.Millisecond)

	if IsRunning() {
		return fmt.Errorf("Claude 未能在超时时间内关闭")
	}

	return nil
}

// Launch 启动 Claude 应用
func Launch() error {
	return exec.Command("open", "-b", bundleID).Run()
}

// LaunchWithDataDir 以独立数据目录启动新的 Claude 实例
func LaunchWithDataDir(dataDir string) error {
	return exec.Command("open", "-na", appName, "--args", "--user-data-dir="+dataDir).Run()
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

	// 先发 SIGTERM 优雅退出
	killInstanceProcs(dataDir, false)

	for range 30 {
		if !IsInstanceRunning(dataDir) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// 超时后强制终止
	killInstanceProcs(dataDir, true)
	time.Sleep(500 * time.Millisecond)

	if IsInstanceRunning(dataDir) {
		return fmt.Errorf("实例未能在超时时间内关闭")
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
