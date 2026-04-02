package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "列出所有 profiles",
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		names, err := pm.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		if len(names) == 0 {
			fmt.Println("暂无 profile，使用 cpm add <name> 添加")
			return
		}

		current := pm.Current()
		for _, name := range names {
			if name == current {
				fmt.Printf("  * %s (当前)\n", name)
			} else {
				fmt.Printf("    %s\n", name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
