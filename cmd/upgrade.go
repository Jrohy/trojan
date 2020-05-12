package cmd

import (
	"github.com/spf13/cobra"
	"trojan/core"
)

// upgradeCmd represents the update command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "升级数据库",
	Run: func(cmd *cobra.Command, args []string) {
		core.GetMysql().UpgradeDB()
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
