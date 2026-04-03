package profile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/i18n"
)

// State persists the currently active profile
type State struct {
	Current string `json:"current"`
}

// Manager handles profile CRUD and switching
type Manager struct {
	baseDir   string
	stateFile string
	state     State
}

// NewManager initializes and returns a Manager
func NewManager() (*Manager, error) {
	baseDir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(baseDir, "profiles"), 0755); err != nil {
		return nil, fmt.Errorf(i18n.T("failed to create config directory: %w", "创建配置目录失败: %w"), err)
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

// BaseDir returns the cpm config root directory
func (m *Manager) BaseDir() string {
	return m.baseDir
}

// ProfileDir returns the storage directory for the given profile
func (m *Manager) ProfileDir(name string) string {
	return m.profileDir(name)
}

// Current returns the name of the active profile
func (m *Manager) Current() string {
	return m.state.Current
}

// Exists checks whether a profile exists
func (m *Manager) Exists(name string) bool {
	_, err := os.Stat(m.profileDir(name))
	return err == nil
}

// List returns all profile names sorted alphabetically
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

// Add saves the current Claude auth data as the named profile
func (m *Manager) Add(name string) error {
	if m.Exists(name) {
		return fmt.Errorf(i18n.T("profile [%s] already exists", "profile [%s] 已存在"), name)
	}

	claudeDir, err := claude.DataDir()
	if err != nil {
		return err
	}

	profDir := m.profileDir(name)
	if err = os.MkdirAll(profDir, 0755); err != nil {
		return fmt.Errorf(i18n.T("failed to create profile directory: %w", "创建 profile 目录失败: %w"), err)
	}

	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(claudeDir, entry)
		dst := filepath.Join(profDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = os.RemoveAll(profDir)
			return fmt.Errorf(i18n.T("failed to copy %s: %w", "复制 %s 失败: %w"), entry, err)
		}
	}

	m.state.Current = name

	return m.saveState()
}

// Switch switches to the named profile
func (m *Manager) Switch(name string) error {
	if !m.Exists(name) {
		return fmt.Errorf(i18n.T("profile [%s] does not exist", "profile [%s] 不存在"), name)
	}

	if m.state.Current == name {
		return fmt.Errorf(i18n.T("already on profile [%s]", "当前已是 profile [%s]"), name)
	}

	claudeDir, err := claude.DataDir()
	if err != nil {
		return err
	}

	// Save current auth data back to the active profile
	if m.state.Current != "" && m.Exists(m.state.Current) {
		currentDir := m.profileDir(m.state.Current)
		for _, entry := range claude.AuthEntries() {
			src := filepath.Join(claudeDir, entry)
			dst := filepath.Join(currentDir, entry)
			_ = copyEntry(src, dst)
		}
	}

	// Restore auth data from the target profile
	targetDir := m.profileDir(name)
	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(targetDir, entry)
		dst := filepath.Join(claudeDir, entry)

		_ = os.RemoveAll(dst)

		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf(i18n.T("failed to restore %s: %w", "恢复 %s 失败: %w"), entry, err)
		}
	}

	m.state.Current = name

	return m.saveState()
}

// Remove deletes the named profile
func (m *Manager) Remove(name string) error {
	if !m.Exists(name) {
		return fmt.Errorf(i18n.T("profile [%s] does not exist", "profile [%s] 不存在"), name)
	}

	if m.state.Current == name {
		m.state.Current = ""
		_ = m.saveState()
	}

	return os.RemoveAll(m.profileDir(name))
}

// Rename renames a profile
func (m *Manager) Rename(oldName, newName string) error {
	if !m.Exists(oldName) {
		return fmt.Errorf(i18n.T("profile [%s] does not exist", "profile [%s] 不存在"), oldName)
	}

	if m.Exists(newName) {
		return fmt.Errorf(i18n.T("profile [%s] already exists", "profile [%s] 已存在"), newName)
	}

	if err := os.Rename(m.profileDir(oldName), m.profileDir(newName)); err != nil {
		return fmt.Errorf(i18n.T("rename failed: %w", "重命名失败: %w"), err)
	}

	if m.state.Current == oldName {
		m.state.Current = newName
		return m.saveState()
	}

	return nil
}

// InstanceDir returns the standalone instance data directory for a profile
func (m *Manager) InstanceDir(name string) string {
	return filepath.Join(m.baseDir, "instances", name)
}

// PrepareInstance sets up the instance directory (copies auth files + syncs shared configs)
func (m *Manager) PrepareInstance(name string) (string, error) {
	if !m.Exists(name) {
		return "", fmt.Errorf(i18n.T("profile [%s] does not exist", "profile [%s] 不存在"), name)
	}

	instDir := m.InstanceDir(name)
	err := os.MkdirAll(instDir, 0755)
	if err != nil {
		return "", fmt.Errorf(i18n.T("failed to create instance directory: %w", "创建实例目录失败: %w"), err)
	}

	var claudeDir string
	claudeDir, err = claude.DataDir()
	if err != nil {
		return "", err
	}

	profDir := m.profileDir(name)

	// Copy auth files
	for _, entry := range claude.AuthEntries() {
		src := filepath.Join(profDir, entry)
		dst := filepath.Join(instDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf(i18n.T("failed to copy auth file %s: %w", "复制认证文件 %s 失败: %w"), entry, err)
		}
	}

	// Sync shared configs
	for _, entry := range claude.SharedConfigs() {
		src := filepath.Join(claudeDir, entry)
		dst := filepath.Join(instDir, entry)
		if err = copyEntry(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf(i18n.T("failed to sync config %s: %w", "同步配置 %s 失败: %w"), entry, err)
		}
	}

	return instDir, nil
}

// copyEntry copies a file or directory
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

// configDir returns the cpm config directory (cross-platform)
func configDir() (string, error) {
	// Windows uses %APPDATA%\cpm
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "cpm"), nil
		}
	}

	// macOS / Linux uses ~/.config/cpm
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}

	return filepath.Join(home, ".config", "cpm"), nil
}
