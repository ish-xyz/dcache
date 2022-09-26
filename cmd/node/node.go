package cmd

import (
	"os"
	"regexp"
	"time"

	"github.com/go-playground/validator"
	"github.com/ish-xyz/dcache/pkg/node"
	"github.com/ish-xyz/dcache/pkg/node/downloader"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	verbose          bool
	insecure         bool // insecure upstream connection
	port             int
	maxConnections   int
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

}

func mapArgs() {
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

func registerNode(nodeObj *node.Node) {
	logrus.Info("registering node... (will retry forever)")
	for !node.Registered {
		nodeObj.Register()
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
		mapArgs()
	}

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if name == "" {
		name, _ = os.Hostname()
		// NOTE: I'm not checking err here,
		// cause if the hostname is empty it would fail
		// during struct validation later
	}

	validate := validator.New()
	nodeObj := node.NewNode(name, ipv4, "http", schedulerAddress, port, maxConnections)
	err := validate.Struct(nodeObj)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	re := regexp.MustCompile(proxyRegex)
	dw := downloader.NewDownloader(
		logger.WithField("component", "downloader"),
	)
	uconf := &node.UpstreamConfig{
		Address:  upstream,
		Insecure: insecure,
	}

	serverObj := *node.NewServer(nodeObj, dataDir, uconf, dw, re)
	err = validate.Struct(serverObj)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	registerNode(nodeObj)
	serverObj.Run()
}
