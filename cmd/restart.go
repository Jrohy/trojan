package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启trojan",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.Restart()
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
