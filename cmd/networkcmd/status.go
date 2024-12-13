// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"github.com/ava-labs/avalanche-cli/pkg/cobrautils"
	"github.com/ava-labs/avalanche-cli/pkg/localnet"
	"github.com/ava-labs/avalanche-cli/pkg/ux"
	"github.com/ava-labs/avalanche-network-runner/server"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Prints the status of the local network",
		Long: `The network status command prints whether or not a local Avalanche
network is running and some basic stats about the network.`,

		RunE: networkStatus,
		Args: cobrautils.ExactArgs(0),
	}
	cmd.Flags().BoolVarP(&asJson, "json", "j", false, "Print json serialized information")

	return cmd
}

func networkStatus(*cobra.Command, []string) error {
	clusterInfo, err := localnet.GetClusterInfo()
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			ux.Logger.PrintToUser("No local network running")
			return nil
		}
		return err
	}
	if clusterInfo != nil {
		ux.Logger.PrintToUser("Network is Up:")
		ux.Logger.PrintToUser("  Number of Nodes: %d", len(clusterInfo.NodeNames))
		ux.Logger.PrintToUser("  Number of Custom VMs: %d", len(clusterInfo.CustomChains))
		ux.Logger.PrintToUser("  Network Healthy: %t", clusterInfo.Healthy)
		ux.Logger.PrintToUser("  Custom VMs Healthy: %t", clusterInfo.CustomChainsHealthy)
		ux.Logger.PrintToUser("")
		if rpcURLs, nodes, err := localnet.GatherEndpoints(""); err != nil {
			return err
		} else if !asJson {
			localnet.PrintEndpoints(ux.Logger.PrintToUser, &rpcURLs, &nodes)
		} else if err = ux.Logger.PrintJSONToUser( map[string]interface{}{"rpc_urls": rpcURLs, "nodes": nodes}); err != nil{
			return err
		}
	} else {
		ux.Logger.PrintToUser("No local network running")
	}

	// TODO: verbose output?
	// ux.Logger.PrintToUser(status.String())

	return nil
}
