package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "显示当前 profile",
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		current := pm.Current()
		if current == "" {
			fmt.Println("当前未设置 profile")
			return
		}

		fmt.Println(current)
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
