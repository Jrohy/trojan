package cmd

import (
	"github.com/spf13/cobra"
	"trojan/web"
)

var port int

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "以web方式启动",
	Run: func(cmd *cobra.Command, args []string) {
		web.Start(port)
	},
}

func init() {
	webCmd.Flags().IntVarP(&port, "port", "p", 80, "web服务启动端口")
	rootCmd.AddCommand(webCmd)
}
