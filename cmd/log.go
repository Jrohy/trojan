package cmd

import (
	"trojan/trojan"

	"github.com/spf13/cobra"
)

var line int

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "查看trojan日志",
	Run: func(cmd *cobra.Command, args []string) {
		trojan.Log(line)
	},
}

func init() {
	logCmd.Flags().IntVarP(&line, "line", "n", 300, "查看日志行数")
	rootCmd.AddCommand(logCmd)
}
