package scheduler

import (
	"os"

	"github.com/go-playground/validator"
	"github.com/ish-xyz/dcache/pkg/scheduler"
	"github.com/ish-xyz/dcache/pkg/scheduler/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config      string
	address     string
	storageType string
	algo        string
	maxProcs    int
	verbose     bool

	Cmd = &cobra.Command{
		Use:   "scheduler",
		Short: "Run dcache scheduler",
		Run:   exec,
	}
)

func CLI() {
	Cmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Config file path")
	Cmd.PersistentFlags().StringVarP(&address, "address", "a", ":8000", "Address of the scheduler")
	Cmd.PersistentFlags().StringVarP(&storageType, "storage-type", "s", "memory", "Backend storage for schedulers")
	Cmd.PersistentFlags().StringVarP(&algo, "algo", "x", "LeastConnections", "Algorithm used by scheduler.")
	Cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Run scheduler in debug mode")

	viper.BindPFlag("scheduler.address", Cmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("scheduler.storage.type", Cmd.PersistentFlags().Lookup("storage-type"))
	viper.BindPFlag("scheduler.algo", Cmd.PersistentFlags().Lookup("algo"))
	viper.BindPFlag("scheduler.verbose", Cmd.PersistentFlags().Lookup("verbose"))
}

func mappping() {
	address = viper.Get("scheduler.address").(string)
	storageType = viper.Get("scheduler.storage.type").(string)
	algo = viper.Get("scheduler.algo").(string)
	verbose = viper.Get("scheduler.verbose").(bool)
	maxProcs = viper.Get("scheduler.maxProcs").(int)

}

func exec(cmd *cobra.Command, args []string) {

	if config != "" {
		viper.SetConfigFile(config)
		err := viper.ReadInConfig()
		if err != nil {
			logrus.Errorf("fatal error reading config file: %w", err)
			os.Exit(101)
		}
		mappping()
	}

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	validate := validator.New()

	store, err := storage.NewStorage(
		viper.Get("scheduler.storage.type").(string),
		map[string]string{}, // TODO: add actual options from CLI
	)
	if err != nil {
		logrus.Errorln(err)
		os.Exit(102)
	}
	sch := scheduler.NewScheduler(
		validate,
		store,
		viper.Get("scheduler.algo").(string),
	)
	srv := scheduler.NewServer(
		viper.Get("scheduler.address").(string),
		sch,
	)
	srv.Run()
}
