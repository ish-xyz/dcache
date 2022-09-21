package cmd

import (
	"os"

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
	rootCmd.AddCommand(schedulerCmd)
	rootCmd.AddCommand(nodeCmd)
	schedulercmd.CLI()
	nodeCLI()
}

func Execute() error {
	return rootCmd.Execute()
}
