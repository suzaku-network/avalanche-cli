// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"fmt"
	"sort"

	"golang.org/x/exp/maps"

	"github.com/ava-labs/avalanche-cli/pkg/utils"
	"github.com/ava-labs/avalanche-network-runner/rpcpb"
	"github.com/ava-labs/avalanche-cli/pkg/models"
)

// GatherLocalNetworkEndpoints gather the endpoints coming from the status call
func GatherEndpoints(
	subnetName string,
) ([]models.RPCURL, []models.Node, error) {
	RPCURLs := []models.RPCURL{}
	nodes := []models.Node{}
	clusterInfo, err := GetClusterInfo()
	if err != nil {
		return nil, nil, err
	}
	for _, chainInfo := range clusterInfo.CustomChains {
		if subnetName == "" || chainInfo.ChainName == subnetName {
			rpcUrl, err := GatherSubnetEndpoints( clusterInfo, chainInfo)
			if err != nil {
				return RPCURLs, nodes, err
			}
			RPCURLs = append(RPCURLs, rpcUrl)
		}
	}
	nodes, err = GatherNetworkEndpoints(clusterInfo)
	if err != nil {
		return RPCURLs, nodes, err
	}
		return RPCURLs, nodes, nil
}

func GatherSubnetEndpoints(
	clusterInfo *rpcpb.ClusterInfo,
	chainInfo *rpcpb.CustomChainInfo, 
) (models.RPCURL, error) {
	rpcURLs := models.RPCURL{}
	nodeInfos := maps.Values(clusterInfo.NodeInfos)
	nodeUris := utils.Map(nodeInfos, func(nodeInfo *rpcpb.NodeInfo) string { return nodeInfo.GetUri() })
	if len(nodeUris) == 0 {
		return rpcURLs, fmt.Errorf("network has no nodes")
	}
	sort.Strings(nodeUris)
	refNodeURI := nodeUris[0]
	nodeInfo := utils.Find(nodeInfos, func(nodeInfo *rpcpb.NodeInfo) bool { return nodeInfo.GetUri() == refNodeURI })
	if nodeInfo == nil {
		return rpcURLs, fmt.Errorf("unexpected nil nodeInfo")
	}
	blockchainIDURL := fmt.Sprintf("%s/ext/bc/%s/rpc", (*nodeInfo).GetUri(), chainInfo.ChainId)
	rpcURLs = models.RPCURL{Name: chainInfo.ChainName, URL: blockchainIDURL}
	if utils.InsideCodespace() {
		var err error
		blockchainIDURL, err = utils.GetCodespaceURL(blockchainIDURL)
		if err != nil {
			return rpcURLs, err
		}
		rpcURLs.CodespaceURL = blockchainIDURL
	}
	return rpcURLs, nil
}

func GatherNetworkEndpoints(
	clusterInfo *rpcpb.ClusterInfo,
) ([]models.Node, error) {
	nodes := []models.Node{}
	insideCodespace := utils.InsideCodespace()
	nodeNames := clusterInfo.NodeNames
	sort.Strings(nodeNames)
	nodeInfos := map[string]*rpcpb.NodeInfo{}
	for _, nodeInfo := range clusterInfo.NodeInfos {
		nodeInfos[nodeInfo.Name] = nodeInfo
	}
	var err error
	for index, nodeName := range nodeNames {
		nodeInfo := nodeInfos[nodeName]
		nodeURL := nodeInfo.GetUri()
		nodes = append(nodes, models.Node{Name: nodeInfo.Name, NodeID: nodeInfo.Id, URL: nodeURL})
		if insideCodespace {
			nodeURL, err = utils.GetCodespaceURL(nodeURL)
			if err != nil {
				return nodes, err
			}
			nodes[index].CodespaceURL = nodeURL
		}
	}
	return nodes, nil
}
