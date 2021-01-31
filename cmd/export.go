package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"trojan/core"
	"trojan/util"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出数据库sql文件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mysql := core.GetMysql()
		if err := mysql.DumpSql(args[0]); err != nil {
			fmt.Println(util.Red(err.Error()))
		} else {
			fmt.Println(util.Green("导出sql成功!"))
		}
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
