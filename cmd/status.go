package cmd

import (
	"trojan/trojan"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看trojan状态",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.Status(true)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
