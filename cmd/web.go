package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"trojan/web"
)

var (
	port int
	ssl  bool
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "以web方式启动",
	Run: func(cmd *cobra.Command, args []string) {
		if ssl && port == 80 {
			fmt.Println("启动web服务失败!")
			fmt.Println("以https方式运行必须传参-p来指定https的运行端口(不能为80)")
			return
		}
		web.Start(port, ssl)
	},
}

func init() {
	webCmd.Flags().IntVarP(&port, "port", "p", 80, "web服务启动端口")
	webCmd.Flags().BoolVarP(&ssl, "ssl", "", false, "web服务是否以https方式运行")
	rootCmd.AddCommand(webCmd)
}
