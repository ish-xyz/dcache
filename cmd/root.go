package cmd

import (
	"os"

	nodecmd "github.com/ish-xyz/dcache/cmd/node"
	schedulercmd "github.com/ish-xyz/dcache/cmd/scheduler"
	"github.com/spf13/cobra"
)

var (
	Version = "unset"
	rootCmd = &cobra.Command{
		Use:   "dcache",
		Short: "Distributed Caching Platform",
		Long:  "Distributed Caching Platform",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(schedulercmd.Cmd)
	rootCmd.AddCommand(nodecmd.Cmd)
	schedulercmd.CLI()
	nodecmd.CLI()
}

func Execute() error {
	return rootCmd.Execute()
}
