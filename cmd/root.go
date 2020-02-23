/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
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
		trojan.InstallTls()
		trojan.InstallMysql()
		trojan.Restart()
	}
}

func mainMenu() {
	check()
exit:
	for {
		fmt.Println()
		fmt.Println(util.Cyan("欢迎使用trojan管理程序"))
		fmt.Println()
		menuList := []string{"trojan管理", "用户管理", "安装管理", "查看配置", "生成客户端配置文件"}
		for i := 0; i < len(menuList); i++ {
			if i%2 == 0 {
				fmt.Printf("%d.%-15s\t", i+1, menuList[i])
			} else {
				fmt.Printf("%d.%-15s\n\n", i+1, menuList[i])
			}
		}
		fmt.Println("\n")
		switch util.LoopInput("请选择: ", menuList, false) {
		case 1:
			trojan.ControllMenu()
		case 2:
			trojan.UserMenu()
		case 3:
			trojan.InstallMenu()
		case 4:
			trojan.UserList()
		case 5:
			trojan.GenClientJson()
		default:
			break exit
		}
	}
}
