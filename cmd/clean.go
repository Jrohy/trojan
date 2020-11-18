package cmd

import (
	"github.com/spf13/cobra"
	"trojan/trojan"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清空指定用户流量",
	Long: `传入指定用户名来清空用户流量, 多个用户名空格隔开, 例如:
trojan clean zhangsan lisi
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		trojan.CleanDataByName(args)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
