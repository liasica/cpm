package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <name>",
	Short: i18n.T("Launch a separate Claude instance with a profile", "以指定 profile 启动独立的 Claude 实例（双开）"),
	Long: i18n.T(
		"Launch a Claude instance with an isolated data directory for simultaneous multi-account usage\n\nMCP config and app settings are synced from the main instance automatically",
		"启动一个使用独立数据目录的 Claude 实例，实现多账户同时在线\n\nMCP 配置和应用设置会自动从主实例同步过来",
	),
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		instDir := pm.InstanceDir(name)
		if claude.IsInstanceRunning(instDir) {
			fmt.Fprintf(os.Stderr, i18n.T("Instance [%s] is already running\n", "profile [%s] 的实例已在运行中\n"), name)
			os.Exit(1)
		}

		fmt.Print(i18n.T("Preparing instance...", "正在准备实例..."))
		instDir, err = pm.PrepareInstance(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(i18n.T(" done", " 完成"))

		fmt.Print(i18n.T("Starting Claude...", "正在启动 Claude..."))
		if err = claude.LaunchWithDataDir(instDir); err != nil {
			fmt.Fprintf(os.Stderr, "\n"+i18n.T("Failed to start: %v\n", "启动失败: %v\n"), err)
			os.Exit(1)
		}
		fmt.Println(i18n.T(" done", " 完成"))

		fmt.Printf(i18n.T("Launched Claude instance [%s]\n", "已启动 Claude 实例 [%s]\n"), name)
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
