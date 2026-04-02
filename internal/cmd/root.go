package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cpm",
	Short: "Claude Profile Manager - 快速切换 Claude 桌面应用账户",
	Long: `cpm 管理 Claude macOS 桌面应用的多账户 profile

只切换认证数据（Cookies/Storage），Claude Code 会话和应用配置保持共享`,
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
			fmt.Println("暂无 profile，使用 cpm add <name> 添加")
			return nil
		}

		current := pm.Current()
		fmt.Println("Profiles:")
		fmt.Println()
		for _, name := range names {
			if name == current {
				fmt.Printf("  * %s (当前)\n", name)
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
