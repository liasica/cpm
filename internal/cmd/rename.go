package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: i18n.T("Rename a profile", "重命名 profile"),
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err = pm.Rename(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf(i18n.T("Renamed [%s] to [%s]\n", "已将 [%s] 重命名为 [%s]\n"), args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
