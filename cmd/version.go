package cmd

import (
	"fmt"
	"trojan/util"

	"github.com/spf13/cobra"
)

var (
	// Version 程序版本号
	Version string
	// BuildDate 编译时间
	BuildDate string
	// GoVersion go版本
	GoVersion string
	// GitVersion git版本
	GitVersion string
)

// versionCmd represents the Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本号",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		fmt.Printf("Version: %s\n\n", util.Cyan(Version))
		fmt.Printf("BuildDate: %s\n\n", util.Cyan(BuildDate))
		fmt.Printf("GoVersion: %s\n\n", util.Cyan(GoVersion))
		fmt.Printf("GitVersion: %s\n\n", util.Cyan(GitVersion))
		fmt.Println(util.ExecCommandWithResult("/usr/bin/trojan/trojan -v"))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
