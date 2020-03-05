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
	completionCmd.AddCommand(&cobra.Command{Use: "bash", Short: "bash命令补全", Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	}})
	completionCmd.AddCommand(&cobra.Command{Use: "zsh", Short: "zsh命令补全", Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenZshCompletion(os.Stdout)
	}})
}
