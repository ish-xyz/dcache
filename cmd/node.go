package cmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

var (
	parentCtx = context.Background()
)

func nodeCLI() {
	return
}

func generateNewID() string {
	return uuid.New().String()
}

func startNode(cmd *cobra.Command, args []string) {

	var requestIDKey node.ContextKey = "X-Request-Id"

	_node := node.NewNode(requestIDKey, "mynode", "127.0.0.1", "http://127.0.0.1:8000", 3000)
	ctx := context.WithValue(parentCtx, requestIDKey, generateNewID())
	_node.Register(ctx)
	fmt.Println(_node.GetStat(ctx))

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
