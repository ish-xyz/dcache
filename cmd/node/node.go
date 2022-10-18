package node

import (
	"os"
	"regexp"
	"time"

	"github.com/ish-xyz/dcache/cmd/utils"
	"github.com/ish-xyz/dcache/pkg/node/client"
	"github.com/ish-xyz/dcache/pkg/node/downloader"
	"github.com/ish-xyz/dcache/pkg/node/notifier"
	"github.com/ish-xyz/dcache/pkg/node/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	verbose        bool
	scheme         = "http"
	insecure       bool // insecure upstream connection
	port           int
	maxConnections int

	gcMaxAtimeAge  string
	gcInterval     string
	gcMaxDiskUsage string

	name             string
	ipv4             string
	dataDir          string
	config           string
	upstream         string
	proxyRegex       string
	schedulerAddress string

	Cmd = &cobra.Command{
		Use:   "node",
		Short: "Run dcache node",
		Run:   exec,
	}
)

func CLI() {
	Cmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Config file path")
	Cmd.PersistentFlags().StringVarP(&name, "name", "n", "", "Name of the node, defaults to hostname")
	Cmd.PersistentFlags().StringVarP(&ipv4, "ip", "i", "", "IPV4 address of the node, that gets advertised to the scheduler")
	Cmd.PersistentFlags().IntVarP(&port, "port", "p", 8100, "Port of the node, that gets advertised to the scheduler")
	Cmd.PersistentFlags().IntVarP(&maxConnections, "max-conns", "m", 10, "Max connections to node")
	Cmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "/var/dcache/data", "Path to the data dir")
	Cmd.PersistentFlags().StringVarP(&upstream, "upstream", "u", "", "URL of the upstream registry")
	Cmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Insecure connection to upstream")
	Cmd.PersistentFlags().StringVarP(&proxyRegex, "proxy-regex", "r", "*blob/sha256*", "Regex for the node proxy")
	Cmd.PersistentFlags().StringVarP(&schedulerAddress, "scheduler-address", "s", "", "Full http url of the scheduler")
	Cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Run node in verbose mode")
	Cmd.PersistentFlags().StringVarP(&gcMaxAtimeAge, "gc-max-atime-age", "t", "12h", "Garbage collector max atime age for files")
	Cmd.PersistentFlags().StringVarP(&gcInterval, "gc-interval", "z", "120m", "Garbage collector interval")
	Cmd.PersistentFlags().StringVarP(&gcMaxDiskUsage, "gc-max-disk-usage", "x", "1G", "Garbage collector max dataDir size (default value 1GB)")

	viper.BindPFlag("node.name", Cmd.PersistentFlags().Lookup("name"))
	viper.BindPFlag("node.ip", Cmd.PersistentFlags().Lookup("ip"))
	viper.BindPFlag("node.port", Cmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("node.dataDir", Cmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("node.upstream.address", Cmd.PersistentFlags().Lookup("upstream"))
	viper.BindPFlag("node.upstream.insecure", Cmd.PersistentFlags().Lookup("insecure"))
	viper.BindPFlag("node.proxy.regex", Cmd.PersistentFlags().Lookup("proxy-regex"))
	viper.BindPFlag("node.scheduler.address", Cmd.PersistentFlags().Lookup("scheduler-address"))
	viper.BindPFlag("node.verbose", Cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("node.gc.maxAtimeAge", Cmd.PersistentFlags().Lookup("gc-max-atime-age"))
	viper.BindPFlag("node.gc.interval", Cmd.PersistentFlags().Lookup("gc-interval"))
	viper.BindPFlag("node.gc.maxDiskUsage", Cmd.PersistentFlags().Lookup("gc-max-disk-usage"))

}

func argumentsMapping() {

	name = viper.Get("node.name").(string)
	ipv4 = viper.Get("node.ip").(string)
	port = viper.Get("node.port").(int)
	verbose = viper.Get("node.verbose").(bool)
	dataDir = viper.Get("node.dataDir").(string)
	insecure = viper.Get("node.upstream.insecure").(bool)
	upstream = viper.Get("node.upstream.address").(string)
	proxyRegex = viper.Get("node.proxy.regex").(string)
	schedulerAddress = viper.Get("node.scheduler.address").(string)
	gcMaxAtimeAge = viper.Get("node.gc.maxAtimeAge").(string)
	gcMaxDiskUsage = viper.Get("node.gc.maxDiskUsage").(string)
	gcInterval = viper.Get("node.gc.interval").(string)

}

func registerNode(c *client.Client) {
	logrus.Info("registering node... (will retry until completed)")
	for !client.Registered {
		c.CreateNode(ipv4, scheme, port, maxConnections)
		time.Sleep(time.Duration(2) * time.Second)
	}
	logrus.Info("registration completed.")
}

func exec(cmd *cobra.Command, args []string) {

	logger := logrus.New()

	if config != "" {
		viper.SetConfigFile(config)
		err := viper.ReadInConfig()
		if err != nil {
			logrus.Errorf("fatal error reading config file: %v", err)
			return
		}
		argumentsMapping()
	}

	if verbose {
		logger.SetLevel(logrus.DebugLevel)
		logrus.SetLevel(logrus.DebugLevel)
	}

	if name == "" {
		name, _ = os.Hostname()
		// NOTE: I'm not checking err here,
		// cause if the hostname is empty it would fail
		// during struct validation later
	}

	gcMaxAtimeAge, err := time.ParseDuration(gcMaxAtimeAge)
	if err != nil {
		logrus.Errorln("failed to parse duration gcMaxAtimeAge")
		os.Exit(101)
	}
	gcInterval, err := time.ParseDuration(gcInterval)
	if err != nil {
		logrus.Errorln("failed to parse duration gcInterval")
		os.Exit(102)
	}
	gcMaxDiskUsage, err := utils.ParseDataSize(gcMaxDiskUsage)
	if err != nil {
		logrus.Errorln("failed to parse data size:", err)
		os.Exit(102)
	}

	nc := client.NewClient(name, schedulerAddress, logger.WithField("component", "node.client"))
	dw := downloader.NewDownloader(
		logger.WithField("component", "node.downloader"),
		dataDir,
		gcMaxAtimeAge,
		gcInterval,
		gcMaxDiskUsage,
	)
	nt := notifier.NewNotifier(
		nc,
		dataDir,
		logger.WithField("component", "node.notifier"),
	)
	srv := server.NewNode(
		nc,
		&server.UpstreamConfig{
			Address:  upstream,
			Insecure: insecure,
		},
		dataDir,
		scheme,
		ipv4,
		port,
		maxConnections,
		dw,
		regexp.MustCompile(proxyRegex),
		logger.WithField("component", "node.server"),
	)

	err = utils.Validate(nc, srv, nt, dw)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		os.Exit(103)
	}

	// Execution
	registerNode(nc)

	logrus.Infoln("starting routines...")
	go dw.Run()
	go nt.Watch()
	go dw.GC.Run()
	srv.Run()
}
