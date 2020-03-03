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
	"trojan/util"

	"github.com/spf13/cobra"
)

const VERSION = "v1.3.1"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本号",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("管理程序版本号: " + util.Cyan(VERSION))
		fmt.Println()
		fmt.Println("trojan版本号: ")
		fmt.Println(util.ExecCommandWithResult("/usr/bin/trojan/trojan -v"))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
