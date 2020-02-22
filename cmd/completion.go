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
	"github.com/spf13/cobra"
	"os"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "自动命令补全(支持bash和zsh)",
	Long: `
支持bash和zsh的命令补全
a. bash环境添加下面命令到 ~/.bashrc 
   source <(trojan completion bash)

b. zsh环境添加以下命令到~/.zshrc
   source <(trojan completion zsh)
`,
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(&cobra.Command{Use:"bash", Short:"bash命令补全", Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	}})
	completionCmd.AddCommand(&cobra.Command{Use:"zsh", Short:"zsh命令补全",  Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenZshCompletion(os.Stdout)
	}})
}
