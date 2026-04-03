package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cpm",
	Short: i18n.T("Claude Profile Manager - quickly switch Claude desktop app accounts", "Claude Profile Manager - 快速切换 Claude 桌面应用账户"),
	Long: i18n.T(
		"cpm manages multiple account profiles for the Claude desktop app\n\nOnly auth data (Cookies/Storage) is switched; Claude Code sessions and app config stay shared",
		"cpm 管理 Claude macOS 桌面应用的多账户 profile\n\n只切换认证数据（Cookies/Storage），Claude Code 会话和应用配置保持共享",
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		pm, err := profile.NewManager()
		if err != nil {
			return err
		}

		names, err := pm.List()
		if err != nil {
			return err
		}

		if len(names) == 0 {
			fmt.Println(i18n.T("No profiles yet, use `cpm add <name>` to create one", "暂无 profile，使用 cpm add <name> 添加"))
			return nil
		}

		current := pm.Current()
		fmt.Println("Profiles:")
		fmt.Println()
		for _, name := range names {
			if name == current {
				fmt.Printf("  * %s (%s)\n", name, i18n.T("current", "当前"))
			} else {
				fmt.Printf("    %s\n", name)
			}
		}

		return nil
	},
}

// SetVersion 设置版本号
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
