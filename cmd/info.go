package cmd

import (
	"trojan/trojan"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "用户信息列表",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.UserList()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
