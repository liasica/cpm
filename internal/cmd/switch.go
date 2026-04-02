package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var noRestart bool

var switchCmd = &cobra.Command{
	Use:     "switch <name>",
	Aliases: []string{"sw"},
	Short:   "切换到指定 profile",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		wasRunning := claude.IsRunning()

		// 切换前关闭 Claude，确保文件不被占用
		if wasRunning {
			fmt.Print("正在关闭 Claude...")
			if err = claude.Quit(); err != nil {
				fmt.Fprintf(os.Stderr, "\n错误: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(" 完成")
		}

		if err = pm.Switch(name); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("已切换到 profile [%s]\n", name)

		// 如果之前在运行且未禁用重启，则自动启动
		if wasRunning && !noRestart {
			fmt.Print("正在启动 Claude...")
			if err = claude.Launch(); err != nil {
				fmt.Fprintf(os.Stderr, "\n启动失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(" 完成")
		}
	},
}

func init() {
	switchCmd.Flags().BoolVar(&noRestart, "no-restart", false, "切换后不自动重启 Claude")
	rootCmd.AddCommand(switchCmd)
}
