package cmd

import (
	"github.com/spf13/cobra"
	"trojan/web"
)

var (
	host string
	port int
	ssl  bool
	cert string
	key  string
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "以web方式启动",
	Run: func(cmd *cobra.Command, args []string) {
		web.Start(host, port, ssl, cert, key)
	},
}

func init() {
	webCmd.Flags().StringVarP(&host, "host", "", "0.0.0.0", "web服务监听地址")
	webCmd.Flags().IntVarP(&port, "port", "p", 80, "web服务启动端口")
	webCmd.Flags().BoolVarP(&ssl, "ssl", "", false, "web服务是否以https方式运行")
	webCmd.Flags().StringVarP(&cert, "cert", "c", "", "https证书文件路径,默认使用trojan的证书")
	webCmd.Flags().StringVarP(&key, "key", "k", "", "https私钥文件路径,默认使用trojan的私钥")
	rootCmd.AddCommand(webCmd)
}
