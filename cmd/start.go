package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动trojan",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.Start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
