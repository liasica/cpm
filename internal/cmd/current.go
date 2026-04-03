package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: i18n.T("Show current profile", "显示当前 profile"),
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		current := pm.Current()
		if current == "" {
			fmt.Println(i18n.T("No active profile", "当前未设置 profile"))
			return
		}

		fmt.Println(current)
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
