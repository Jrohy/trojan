package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// delCmd represents the del command
var delCmd = &cobra.Command{
	Use:   "del",
	Short: "删除用户",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.DelUser()
	},
}

func init() {
	rootCmd.AddCommand(delCmd)
}
