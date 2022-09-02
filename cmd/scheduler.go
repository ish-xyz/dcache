package cmd

import (
	"fmt"

	"github.com/go-playground/validator"
	"github.com/ish-xyz/dreg/pkg/scheduler"
	"github.com/ish-xyz/dreg/pkg/scheduler/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	schedulerConfig   string
	schedulerAddress  string
	schedulerStorage  string
	schedulerAlgo     string
	schedulerMaxProcs int
	schedulerDebug    bool

	schedulerCmd = &cobra.Command{
		Use:   "scheduler",
		Short: "Run dreg scheduler",
		Run:   startScheduler,
	}
)

func schedulerCLI() {
	schedulerCmd.PersistentFlags().StringVarP(&schedulerConfig, "config", "c", "", "Config file path")
	schedulerCmd.PersistentFlags().StringVarP(&schedulerAddress, "address", "a", ":8000", "Address of the scheduler")
	schedulerCmd.PersistentFlags().StringVarP(&schedulerStorage, "storage-type", "s", "memory", "Backend storage for schedulers")
	schedulerCmd.PersistentFlags().StringVarP(&schedulerAlgo, "algo", "x", "LeastConnections", "Algorithm used by scheduler.")
	schedulerCmd.PersistentFlags().IntVarP(&schedulerMaxProcs, "max-procs", "m", 10, "Max amount of concurrent connections for nodes.")
	schedulerCmd.PersistentFlags().BoolVarP(&schedulerDebug, "debug", "d", false, "Run in debug mode")

	viper.BindPFlag("scheduler.address", schedulerCmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("scheduler.storage.type", schedulerCmd.PersistentFlags().Lookup("storage-type"))
	viper.BindPFlag("scheduler.algo", schedulerCmd.PersistentFlags().Lookup("algo"))
	viper.BindPFlag("scheduler.maxProcs", schedulerCmd.PersistentFlags().Lookup("max-procs"))
	viper.BindPFlag("scheduler.debug", schedulerCmd.PersistentFlags().Lookup("debug"))
}

func startScheduler(cmd *cobra.Command, args []string) {

	var storageOpts map[string]string

	if schedulerDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if schedulerConfig != "" {
		viper.SetConfigFile(schedulerConfig)
		err := viper.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	validate := validator.New()

	store := storage.NewStorage(
		viper.Get("scheduler.storage.type").(string),
		storageOpts,
	)
	sch := scheduler.NewScheduler(
		validate,
		store,
		viper.Get("scheduler.maxProcs").(int),
		viper.Get("scheduler.algo").(string),
	)
	srv := scheduler.NewServer(
		viper.Get("scheduler.address").(string),
		sch,
	)
	srv.Run()
}

/*
===============
=> config.yaml |
===============
scheduler:
  address: ":8000"
  maxProcs: 10
  algo: leastConnections
  storage:
    type: memory/redis

// TO ADD:
  redis:
	address:
	username:
	password:
	tls:
tls: {}
*/