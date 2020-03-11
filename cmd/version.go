package cmd

import (
	"fmt"
	"trojan/trojan"
	"trojan/util"

	"github.com/spf13/cobra"
)

// versionCmd represents the Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本号",
	Run: func(cmd *cobra.Command, args []string) {
		runTime := trojan.RunTime()
		trojanVersion := trojan.Version()
		fmt.Println()
		fmt.Printf("Version: %s\n\n", util.Cyan(trojan.MVersion))
		fmt.Printf("BuildDate: %s\n\n", util.Cyan(trojan.BuildDate))
		fmt.Printf("GoVersion: %s\n\n", util.Cyan(trojan.GoVersion))
		fmt.Printf("GitVersion: %s\n\n", util.Cyan(trojan.GitVersion))
		fmt.Printf("TrojanVersion: %s\n\n", util.Cyan(trojanVersion))
		fmt.Printf("TrojanRunTime: %s\n\n", util.Cyan(runTime))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
