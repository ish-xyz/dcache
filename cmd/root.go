package cmd

import (
	"os"

	nodecmd "github.com/ish-xyz/dreg/cmd/node"
	schedulercmd "github.com/ish-xyz/dreg/cmd/scheduler"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "dreg",
		Short: "A tool to run distributed docker registries",
		Long:  "A tool to run distributed docker registries",
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
