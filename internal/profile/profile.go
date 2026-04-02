package profile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/liasica/cpm/internal/claude"
)

// State 持久化当前活跃的 profile 信息
type State struct {
	Current string `json:"current"`
}

// Manager 负责 profile 的增删改查和切换
type Manager struct {
	baseDir   string
	stateFile string
	state     State
}

// NewManager 初始化并返回 Manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	baseDir := filepath.Join(home, ".config", "cpm")
	if err = os.MkdirAll(filepath.Join(baseDir, "profiles"), 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	m := &Manager{
		baseDir:   baseDir,
		stateFile: filepath.Join(baseDir, "state.json"),
	}
	m.loadState()

	return m, nil
}

func (m *Manager) loadState() {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &m.state)
}

func (m *Manager) saveState() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile, data, 0644)
}

func (m *Manager) profileDir(name string) string {
	return filepath.Join(m.baseDir, "profiles", name)
}

// Current 返回当前 profile 名称
func (m *Manager) Current() string {
	return m.state.Current
}

// Exists 判断 profile 是否存在
func (m *Manager) Exists(name string) bool {
	_, err := os.Stat(m.profileDir(name))
	return err == nil
}

// List 返回所有 profile 名称（按字母排序）
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(m.baseDir, "profiles"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	return names, nil
}

// Add 将当前 Claude 的认证数据保存为指定 profile
func (m *Manager) Add(name string) error {
	if m.Exists(name) {
		return fmt.Errorf("profile [%s] 已存在", name)
	}

	claudeDir, err := claude.DataDir()
	if err != nil {
		return err
	}

	profDir := m.profileDir(name)
	if err = os.MkdirAll(profDir, 0755); err != nil {
		return fmt.Errorf("创建 profile 目录失败: %w", err)
	}

	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(claudeDir, entry)
		dst := filepath.Join(profDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = os.RemoveAll(profDir)
			return fmt.Errorf("复制 %s 失败: %w", entry, err)
		}
	}

	m.state.Current = name

	return m.saveState()
}

// Switch 切换到指定 profile
func (m *Manager) Switch(name string) error {
	if !m.Exists(name) {
		return fmt.Errorf("profile [%s] 不存在", name)
	}

	if m.state.Current == name {
		return fmt.Errorf("当前已是 profile [%s]", name)
	}

	claudeDir, err := claude.DataDir()
	if err != nil {
		return err
	}

	// 将当前认证数据回写到当前 profile
	if m.state.Current != "" && m.Exists(m.state.Current) {
		currentDir := m.profileDir(m.state.Current)
		for _, entry := range claude.AuthEntries() {
			src := filepath.Join(claudeDir, entry)
			dst := filepath.Join(currentDir, entry)
			_ = copyEntry(src, dst)
		}
	}

	// 从目标 profile 恢复认证数据
	targetDir := m.profileDir(name)
	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(targetDir, entry)
		dst := filepath.Join(claudeDir, entry)

		_ = os.RemoveAll(dst)

		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("恢复 %s 失败: %w", entry, err)
		}
	}

	m.state.Current = name

	return m.saveState()
}

// Remove 删除指定 profile
func (m *Manager) Remove(name string) error {
	if !m.Exists(name) {
		return fmt.Errorf("profile [%s] 不存在", name)
	}

	if m.state.Current == name {
		m.state.Current = ""
		_ = m.saveState()
	}

	return os.RemoveAll(m.profileDir(name))
}

// Rename 重命名 profile
func (m *Manager) Rename(oldName, newName string) error {
	if !m.Exists(oldName) {
		return fmt.Errorf("profile [%s] 不存在", oldName)
	}

	if m.Exists(newName) {
		return fmt.Errorf("profile [%s] 已存在", newName)
	}

	if err := os.Rename(m.profileDir(oldName), m.profileDir(newName)); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}

	if m.state.Current == oldName {
		m.state.Current = newName
		return m.saveState()
	}

	return nil
}

// InstanceDir 返回指定 profile 的独立实例数据目录
func (m *Manager) InstanceDir(name string) string {
	return filepath.Join(m.baseDir, "instances", name)
}

// PrepareInstance 为指定 profile 准备独立实例目录（复制认证文件 + 同步共享配置）
func (m *Manager) PrepareInstance(name string) (string, error) {
	if !m.Exists(name) {
		return "", fmt.Errorf("profile [%s] 不存在", name)
	}

	instDir := m.InstanceDir(name)
	err := os.MkdirAll(instDir, 0755)
	if err != nil {
		return "", fmt.Errorf("创建实例目录失败: %w", err)
	}

	var claudeDir string
	claudeDir, err = claude.DataDir()
	if err != nil {
		return "", err
	}

	profDir := m.profileDir(name)

	// 复制认证文件
	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(profDir, entry)
		dst := filepath.Join(instDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("复制认证文件 %s 失败: %w", entry, err)
		}
	}

	// 同步共享配置
	for _, entry := range claude.SharedConfigs() {
		src := filepath.Join(claudeDir, entry)
		dst := filepath.Join(instDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("同步配置 %s 失败: %w", entry, err)
		}
	}

	return instDir, nil
}

// copyEntry 复制文件或目录
func copyEntry(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}

	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var info os.FileInfo
	info, err = in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	_ = os.RemoveAll(dst)
	if err = os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if err = copyEntry(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}
