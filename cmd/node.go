package cmd

import (
	"os"
	"regexp"

	"github.com/go-playground/validator"
	"github.com/ish-xyz/dreg/pkg/node"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	requestIDKey         node.ContextKey = "X-Request-Id"
	nodeVerbose          bool
	nodeInsecureP        bool // insecure upstream connection
	nodeName             string
	nodeIPv4             string
	nodePort             int
	nodeDataDir          string
	nodeConfig           string
	nodeUpstream         string
	nodeProxyRegex       string
	nodeSchedulerAddress string

	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run dreg node",
		Run:   startNode,
	}
)

func nodeCLI() {
	nodeCmd.PersistentFlags().StringVarP(&nodeConfig, "config", "c", "", "Config file path")
	nodeCmd.PersistentFlags().StringVarP(&nodeName, "name", "n", "", "Name of the node, defaults to hostname")
	nodeCmd.PersistentFlags().StringVarP(&nodeIPv4, "ipv4", "i", "", "IPV4 address of the node, that gets advertised to the scheduler")
	nodeCmd.PersistentFlags().IntVarP(&nodePort, "port", "p", 8100, "Port of the node, that gets advertised to the scheduler")
	nodeCmd.PersistentFlags().StringVarP(&nodeDataDir, "data-dir", "d", "/var/dreg/data", "Path to the data dir")
	nodeCmd.PersistentFlags().StringVarP(&nodeUpstream, "upstream", "u", "", "URL of the upstream registry")
	nodeCmd.PersistentFlags().BoolVarP(&nodeInsecureP, "insecure", "k", false, "Insecure connection to upstream")
	nodeCmd.PersistentFlags().StringVarP(&nodeProxyRegex, "proxy-regex", "r", "*blob/sha256*", "Regex for the node proxy")
	nodeCmd.PersistentFlags().StringVarP(&nodeSchedulerAddress, "scheduler-address", "s", "", "Full http url of the scheduler")
	nodeCmd.PersistentFlags().BoolVarP(&nodeVerbose, "verbose", "v", false, "Run node in debug mode")

	viper.BindPFlag("node.name", schedulerCmd.PersistentFlags().Lookup("name"))
	viper.BindPFlag("node.ipv4", schedulerCmd.PersistentFlags().Lookup("ipv4"))
	viper.BindPFlag("node.port", schedulerCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("node.dataDir", schedulerCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("node.upstream.address", schedulerCmd.PersistentFlags().Lookup("upstream"))
	viper.BindPFlag("node.upstream.insecure", schedulerCmd.PersistentFlags().Lookup("insecure"))
	viper.BindPFlag("node.proxy.regex", schedulerCmd.PersistentFlags().Lookup("proxy-regex"))
	viper.BindPFlag("node.scheduler.address", schedulerCmd.PersistentFlags().Lookup("scheduler-address"))
	viper.BindPFlag("node.debug", schedulerCmd.PersistentFlags().Lookup("debug"))

}

func startNode(cmd *cobra.Command, args []string) {

	validate := validator.New()

	if nodeVerbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if nodeConfig != "" {
		viper.SetConfigFile(nodeConfig)
		err := viper.ReadInConfig()
		if err != nil {
			logrus.Errorf("fatal error config file: %w", err)
			return
		}
	}

	if nodeName == "" {
		nodeName, _ = os.Hostname()
		// not checking error here,
		// cause if empty it would fail
		// in struct validation later
	}

	_node := node.NewNode(requestIDKey, nodeName, nodeIPv4, "http", nodeSchedulerAddress, nodePort)

	err := validate.Struct(_node)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	re := regexp.MustCompile(nodeProxyRegex)

	_server := &node.Server{
		Node:    _node,
		DataDir: nodeDataDir,
		Upstream: &node.UpstreamConfig{
			Address:  nodeUpstream,
			Insecure: nodeInsecureP,
		},
		Regex: re,
	}

	err = validate.Struct(_server)
	if err != nil {
		logrus.Errorf("Error while validating user inputs or configuration file")
		logrus.Debugln(err)
		return
	}

	_server.Run()

}
