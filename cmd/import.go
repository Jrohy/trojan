package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"trojan/core"
	"trojan/util"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "导入数据库sql文件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mysql := core.GetMysql()
		if err := mysql.ExecSql(args[0]); err != nil {
			fmt.Println(util.Red(err.Error()))
		} else {
			fmt.Println(util.Green("导入sql成功!"))
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
