package cmd

import (
	"github.com/spf13/cobra"
	"trojan/web"
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "以web形式运行对外暴露api数据",
	Run: func(cmd *cobra.Command, args []string) {
		web.Start()
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
