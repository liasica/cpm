package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "保存当前 Claude 登录为 profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		if err = pm.Add(name); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("已保存当前登录为 profile [%s]\n", name)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
