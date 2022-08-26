package cmd

import (
	"regexp"

	"github.com/ish-xyz/dreg/pkg/node"
	"github.com/spf13/cobra"
)

var (
	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run dreg node",
		Run:   startNode,
	}
)

func nodeCLI() {
	return
}

func startNode(cmd *cobra.Command, args []string) {

	re, _ := regexp.Compile(".*ciao.*")
	proxy := &node.Proxy{
		Node: &node.Node{
			Name: "mynode",
		},
		Upstream: "https://google.com",
		Address:  ":6000",
		Regex:    re,
		IPv4:     "127.0.0.1",
	}

	proxy.Run()
}
