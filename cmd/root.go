package cmd

import (
	"os"

	"github.com/ish-xyz/dreg/pkg/scheduler"
	"github.com/ish-xyz/dreg/pkg/scheduler/storage"
	"github.com/spf13/cobra"
)

var (
	// Flags
	address string

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

	schedulerCmd = &cobra.Command{
		Use:   "scheduler",
		Short: "Run dreg scheduler",
		Long:  "Run dreg scheduler",
		Run:   execScheduler,
	}
)

func init() {
	rootCmd.AddCommand(schedulerCmd)
	schedulerCmd.PersistentFlags().StringVarP(&address, "address", "a", ":8000", "Address of the scheduler")
}

func Execute() error {
	return rootCmd.Execute()
}

func execScheduler(cmd *cobra.Command, args []string) {
	var storageOpts map[string]string
	store := storage.NewStorage("memory", storageOpts)
	sch := scheduler.NewScheduler(store, 20)
	srv := scheduler.NewServer(address, sch)
	srv.Run()
}
