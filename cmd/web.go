package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"trojan/util"
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
	webCmd.AddCommand(&cobra.Command{Use: "stop", Short: "停止trojan-web", Run: func(cmd *cobra.Command, args []string) {
		util.SystemctlStop("trojan-web")
	}})
	webCmd.AddCommand(&cobra.Command{Use: "start", Short: "启动trojan-web", Run: func(cmd *cobra.Command, args []string) {
		util.SystemctlStart("trojan-web")
	}})
	webCmd.AddCommand(&cobra.Command{Use: "restart", Short: "重启trojan-web", Run: func(cmd *cobra.Command, args []string) {
		util.SystemctlRestart("trojan-web")
	}})
	webCmd.AddCommand(&cobra.Command{Use: "status", Short: "查看trojan-web状态", Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(util.SystemctlStatus("trojan-web"))
	}})
	webCmd.AddCommand(&cobra.Command{Use: "log", Short: "查看trojan-web日志", Run: func(cmd *cobra.Command, args []string) {
		util.Log("trojan-web", 300)
	}})
	rootCmd.AddCommand(webCmd)
}
