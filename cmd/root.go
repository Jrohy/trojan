package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"trojan/core"
	"trojan/trojan"
	"trojan/util"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "trojan",
	Run: func(cmd *cobra.Command, args []string) {
		mainMenu()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func check() {
	if !util.IsExists("/usr/local/etc/trojan/config.json") {
		fmt.Println("本机未安装trojan, 正在自动安装...")
		trojan.InstallTrojan()
		core.WritePassword(nil)
		trojan.InstallTls()
		trojan.InstallMysql()
		util.ExecCommand("systemctl restart trojan-web")
	}
}

func mainMenu() {
	check()
exit:
	for {
		fmt.Println()
		fmt.Println(util.Cyan("欢迎使用trojan管理程序"))
		fmt.Println()
		menuList := []string{"trojan管理", "用户管理", "安装管理", "web管理", "查看配置", "生成json"}
		for i := 0; i < len(menuList); i++ {
			if i%2 == 0 {
				fmt.Printf("%d.%-15s\t", i+1, menuList[i])
			} else {
				fmt.Printf("%d.%-15s\n\n", i+1, menuList[i])
			}
		}
		switch util.LoopInput("请选择: ", menuList, false) {
		case 1:
			trojan.ControllMenu()
		case 2:
			trojan.UserMenu()
		case 3:
			trojan.InstallMenu()
		case 4:
			trojan.WebMenu()
		case 5:
			trojan.UserList()
		case 6:
			trojan.GenClientJson()
		default:
			break exit
		}
	}
}
