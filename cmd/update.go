package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// upgradeCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新trojan",
	Long:  "可添加版本号更新特定版本, 比如'trojan update v0.10.0', 不添加版本号则安装最新版",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := ""
		if len(args) == 1 {
			version = args[0]
		}
		trojan.InstallTrojan(version)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
