package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var noRestart bool

var switchCmd = &cobra.Command{
	Use:     "switch <name>",
	Aliases: []string{"sw"},
	Short:   i18n.T("Switch to a profile", "切换到指定 profile"),
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		wasRunning := claude.IsRunning()

		// Close Claude before switching to avoid file lock conflicts
		if wasRunning {
			fmt.Print(i18n.T("Closing Claude...", "正在关闭 Claude..."))
			if err = claude.Quit(); err != nil {
				fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(i18n.T(" done", " 完成"))
		}

		if err = pm.Switch(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf(i18n.T("Switched to profile [%s]\n", "已切换到 profile [%s]\n"), name)

		// Relaunch if it was running and restart is not disabled
		if wasRunning && !noRestart {
			fmt.Print(i18n.T("Starting Claude...", "正在启动 Claude..."))
			if err = claude.Launch(); err != nil {
				fmt.Fprintf(os.Stderr, "\n"+i18n.T("Failed to start: %v\n", "启动失败: %v\n"), err)
				os.Exit(1)
			}
			fmt.Println(i18n.T(" done", " 完成"))
		}
	},
}

func init() {
	switchCmd.Flags().BoolVar(&noRestart, "no-restart", false, i18n.T("don't restart Claude after switching", "切换后不自动重启 Claude"))
	rootCmd.AddCommand(switchCmd)
}
