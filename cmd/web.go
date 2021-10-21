package cmd

import (
	"github.com/spf13/cobra"
	"trojan/web"
)

var (
	host    string
	port    int
	ssl     bool
	timeout int
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "以web方式启动",
	Run: func(cmd *cobra.Command, args []string) {
		web.Start(host, port, timeout, ssl)
	},
}

func init() {
	webCmd.Flags().StringVarP(&host, "host", "", "0.0.0.0", "web服务监听地址")
	webCmd.Flags().IntVarP(&port, "port", "p", 80, "web服务启动端口")
	webCmd.Flags().BoolVarP(&ssl, "ssl", "", false, "web服务是否以https方式运行")
	webCmd.Flags().IntVarP(&timeout, "timeout", "t", 120, "登录超时时间(min)")
	rootCmd.AddCommand(webCmd)
}
