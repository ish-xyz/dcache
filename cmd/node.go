package cmd

import (
	"context"
	"regexp"

	"github.com/ish-xyz/dreg/pkg/node"
	"github.com/sirupsen/logrus"
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

// func generateNewID() string {
// 	return uuid.New().String()
// }

func startNode(cmd *cobra.Command, args []string) {

	// TODO: change with debug flag
	if true {
		logrus.SetLevel(logrus.DebugLevel)
	}

	var requestIDKey node.ContextKey = "X-Request-Id"

	_node := node.NewNode(requestIDKey, "mynode", "127.0.0.1", "http://127.0.0.1:8000", 8100)
	//ctx := context.WithValue(parentCtx, requestIDKey, generateNewID())

	re := regexp.MustCompile(".*asdasdasdasdas.*")
	srv := &node.Server{
		Node:    _node,
		DataDir: "/Users/ishamaraia/repos/dreg/data/",
		Upstream: &node.UpstreamConfig{
			Address:  "https://ish-ar.io",
			Insecure: true,
		},
		Address: ":8100",
		Regex:   re,
	}

	srv.Run()
}
