package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "添加用户",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.AddUser()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
