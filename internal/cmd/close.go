package cmd

import (
	"fmt"
	"os"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <name>",
	Short: "关闭指定 profile 的独立实例",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		instDir := pm.InstanceDir(name)
		if !claude.IsInstanceRunning(instDir) {
			fmt.Printf("profile [%s] 的实例未在运行\n", name)
			return
		}

		fmt.Print("正在关闭实例...")
		if err = claude.CloseInstance(instDir); err != nil {
			fmt.Fprintf(os.Stderr, "\n错误: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(" 完成")
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)
}
