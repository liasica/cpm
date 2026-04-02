package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "以指定 profile 启动独立的 Claude 实例（双开）",
	Long: `启动一个使用独立数据目录的 Claude 实例，实现多账户同时在线

MCP 配置和应用设置会自动从主实例同步过来`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		instDir := pm.InstanceDir(name)
		if claude.IsInstanceRunning(instDir) {
			fmt.Fprintf(os.Stderr, "profile [%s] 的实例已在运行中\n", name)
			os.Exit(1)
		}

		fmt.Print("正在准备实例...")
		instDir, err = pm.PrepareInstance(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n错误: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(" 完成")

		fmt.Print("正在启动 Claude...")
		if err = claude.LaunchWithDataDir(instDir); err != nil {
			fmt.Fprintf(os.Stderr, "\n启动失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(" 完成")

		fmt.Printf("已启动 Claude 实例 [%s]\n", name)
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
