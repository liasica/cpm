package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   i18n.T("List all profiles", "列出所有 profiles"),
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		names, err := pm.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(names) == 0 {
			fmt.Println(i18n.T("No profiles yet, use `cpm add <name>` to create one", "暂无 profile，使用 cpm add <name> 添加"))
			return
		}

		current := pm.Current()
		for _, name := range names {
			if name == current {
				fmt.Printf("  * %s (%s)\n", name, i18n.T("current", "当前"))
			} else {
				fmt.Printf("    %s\n", name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
