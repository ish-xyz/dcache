package cmd

import (
	"fmt"

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

	_node := node.NewNode("mynode", "127.0.0.1", "http://127.0.0.1:8000", 3000)
	_node.Register()

	stat, err := _node.GetStat()
	fmt.Println(stat, err)

	_node.AddNodeConnection()

	stat, err = _node.GetStat()
	fmt.Println(stat, err)

	// re, _ := regexp.Compile(".*ciao.*")
	// proxy := &node.Proxy{
	// 	Node: _node,
	// 	Upstream: &node.UpstreamConfig{
	// 		Address:  "http://ish-ar.io/",
	// 		Insecure: true,
	// 	},
	// 	Address: ":6000",
	// 	Regex:   re,
	// }

	// proxy.Run()
}
