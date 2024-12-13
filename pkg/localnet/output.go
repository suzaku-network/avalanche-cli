// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"fmt"

	"github.com/ava-labs/avalanche-cli/pkg/models"
	"github.com/ava-labs/avalanche-cli/pkg/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// PrintLocalNetworkEndpoints prints the endpoints coming from the status call
func PrintEndpoints(
	printFunc func(msg string, args ...interface{}),
	rpcURLs *[]models.RPCURL, 
	nodes *[]models.Node,
) {
	for _, rpcurl := range *rpcURLs {
		PrintSubnetEndpoints(printFunc, rpcurl)
			printFunc("")
	}
	PrintNetworkEndpoints(printFunc, *nodes)
}

func PrintSubnetEndpoints(
	printFunc func(msg string, args ...interface{}),
	rpcurl models.RPCURL,
) {
	
	t := table.NewWriter()
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.Style().Options.SeparateRows = true
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true},
	})
	t.SetTitle(fmt.Sprintf("%s RPC URLs", rpcurl.Name))
	t.AppendRow(table.Row{"Localhost", rpcurl.URL})
	if rpcurl.CodespaceURL != "" {
		t.AppendRow(table.Row{"Codespace", rpcurl.CodespaceURL})
	}
	printFunc(t.Render())
}

func PrintNetworkEndpoints(
	printFunc func(msg string, args ...interface{}),
	nodes []models.Node,
) {
	t := table.NewWriter()
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.Style().Options.SeparateRows = true
	t.SetTitle("Nodes")
	header := table.Row{"Name", "Node ID", "Localhost Endpoint"}
	insideCodespace := utils.InsideCodespace()
	if insideCodespace {
		header = append(header, "Codespace Endpoint")
	}
	t.AppendHeader(header)
	for _, node := range nodes {
		row := table.Row{node.Name, node.NodeID, node.URL}
		if insideCodespace {
			row = append(row, node.CodespaceURL)
		}
		t.AppendRow(row)
	}
	printFunc(t.Render())
}
