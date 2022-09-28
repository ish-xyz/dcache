package cmd

import (
	"os"
	"regexp"
	"time"

	"github.com/go-playground/validator"
	"github.com/ish-xyz/dcache/pkg/node"
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
	gcMaxDiskUsage int
	gcMinDiskFree  int

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
	Cmd.PersistentFlags().StringVarP(&ipv4, "ipv4", "i", "", "IPV4 address of the node, that gets advertised to the scheduler")
	Cmd.PersistentFlags().IntVarP(&port, "port", "p", 8100, "Port of the node, that gets advertised to the scheduler")
	Cmd.PersistentFlags().IntVarP(&maxConnections, "max-conns", "m", 10, "Max connections to node")
	Cmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "/var/dcache/data", "Path to the data dir")
	Cmd.PersistentFlags().StringVarP(&upstream, "upstream", "u", "", "URL of the upstream registry")
	Cmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Insecure connection to upstream")
	Cmd.PersistentFlags().StringVarP(&proxyRegex, "proxy-regex", "r", "*blob/sha256*", "Regex for the node proxy")
	Cmd.PersistentFlags().StringVarP(&schedulerAddress, "scheduler-address", "s", "", "Full http url of the scheduler")
	Cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Run node in verbose mode")

	viper.BindPFlag("node.name", Cmd.PersistentFlags().Lookup("name"))
	viper.BindPFlag("node.ipv4", Cmd.PersistentFlags().Lookup("ipv4"))
	viper.BindPFlag("node.port", Cmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("node.dataDir", Cmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("node.upstream.address", Cmd.PersistentFlags().Lookup("upstream"))
	viper.BindPFlag("node.upstream.insecure", Cmd.PersistentFlags().Lookup("insecure"))
	viper.BindPFlag("node.proxy.regex", Cmd.PersistentFlags().Lookup("proxy-regex"))
	viper.BindPFlag("node.scheduler.address", Cmd.PersistentFlags().Lookup("scheduler-address"))
	viper.BindPFlag("node.verbose", Cmd.PersistentFlags().Lookup("verbose"))

	viper.BindPFlag("node.gc.maxAtimeAge", Cmd.PersistentFlags().Lookup("gc-max-atime-age"))
	viper.BindPFlag("node.gc.interval", Cmd.PersistentFlags().Lookup("gc-interval"))
	//viper.BindPFlag("node.gc.maxStorage", Cmd.PersistentFlags().Lookup("gc-max-disk-usage"))
	//viper.BindPFlag("node.gc.minDiskFree", Cmd.PersistentFlags().Lookup("gc-min-disk-free"))

}

func argumentsMapping() {
	name = viper.Get("node.name").(string)
	ipv4 = viper.Get("node.ipv4").(string)
	port = viper.Get("node.port").(int)
	verbose = viper.Get("node.verbose").(bool)
	dataDir = viper.Get("node.dataDir").(string)
	insecure = viper.Get("node.upstream.insecure").(bool)
	upstream = viper.Get("node.upstream.address").(string)
	proxyRegex = viper.Get("node.proxy.regex").(string)
	schedulerAddress = viper.Get("node.scheduler.address").(string)
}

func registerNode(client *node.Client) {
	logrus.Info("registering node... (will retry forever)")
	for !node.Registered {
		client.Register(ipv4, scheme, port, maxConnections)
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
			logrus.Errorf("fatal error reading config file: %w", err)
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

	validate := validator.New()
	client := node.NewClient(
		name,
		schedulerAddress,
		logger.WithField("component", "node.client"),
	)
	err := validate.Struct(client)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	dw := downloader.NewDownloader(
		logger.WithField("component", "node.downloader"),
		dataDir,
		time.Duration(30),
		time.Duration(5),
		gcMaxDiskUsage,
		gcMinDiskFree,
	)

	nt := notifier.NewNotifier(
		client,
		dataDir,
		logger.WithField("component", "node.notifier"),
	)

	//TODO: add nt & dw validation
	re := regexp.MustCompile(proxyRegex)
	uconf := &server.UpstreamConfig{
		Address:  upstream,
		Insecure: insecure,
	}
	nodeObj := server.NewNode(
		client,
		uconf,
		dataDir,
		scheme,
		ipv4,
		port,
		maxConnections,
		dw,
		re,
		logger.WithField("component", "node.server"),
	)
	err = validate.Struct(nodeObj)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	// Run programs
	registerNode(client)

	logrus.Infoln("starting daemons...")
	go dw.Run()
	go nt.Watch()
	go dw.GC.Run()
	nodeObj.Run()
}
