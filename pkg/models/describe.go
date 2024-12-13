// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"github.com/ethereum/go-ethereum/common"
)

type SubnetInfo struct {
	Name              string                  `json:"name"`
	VMVersion         string                  `json:"vm_version"`
	VMID              string                  `json:"vm_id"`
	Validation        ValidatorManagementType `json:"validation"`
	LocalNetworks     *[]LocalNetwork          `json:"local_networks,omitempty"`
	Token             *Token                   `json:"token"`
	TokenAllocations  *[]TokenAlloc            `json:"token_allocations"`
	SmartContracts    *[]SmartContract         `json:"smart_contracts"`
	PrecompileConfigs *[]Precompile            `json:"precompile_configs"`
	RPCURLs           *[]RPCURL                `json:"rpc_urls,omitempty"`
	Nodes             *[]Node                  `json:"nodes,omitempty"`
	Wallet            *Wallet                  `json:"wallet,omitempty"`
}

type LocalNetwork struct {
	Name             string      `json:"name,omitempty"`
	ChainID          string      `json:"chain_id,omitempty"`
	SubnetID         string      `json:"subnet_id,omitempty"`
	BlockchainIDCB58 string      `json:"blockchain_id_cb58,omitempty"`
	BlockchainIDHex  string      `json:"blockchain_id_hex,omitempty"`
	RPCEndpoint      string      `json:"rpc_endpoint,omitempty"`
	ChainOwners		   *ChainOwners `json:"owner,omitempty"`
	Teleporter       *Teleporter  `json:"teleporter,omitempty"`
}

type ChainOwners struct {
	Names			[]string	`json:"name,omitempty"`
	Threshold	uint32		`json:"threshold,omitempty"`
}

type Teleporter struct {
	MessengerAddress string `json:"messenger_address,omitempty"`
	RegistryAddress  string `json:"registry_address,omitempty"`
}

type Token struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type TokenAlloc struct {
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Address       string `json:"address,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	AmountToken   string `json:"amount_ash,omitempty"`
	AmountWEI     string `json:"amount_wei,omitempty"`
}

type SmartContract struct {
	Description string `json:"description,omitempty"`
	Address     string `json:"address,omitempty"`
	Deployer    string `json:"deployer,omitempty"`
}

type Precompile struct {
	Name             string           `json:"precompile,omitempty"`
	AdminAddresses   []common.Address `json:"admin_addresses,omitempty"`
	ManagerAddresses []common.Address `json:"manager_addresses,omitempty"`
	EnabledAddresses []common.Address `json:"enabled_addresses,omitempty"`
}

type RPCURL struct {
	Name         string `json:"name,omitempty"`
	URL	         string `json:"url,omitempty"`
	CodespaceURL string `json:"codespace_url,omitempty"`
}

type Node struct {
	Name              string `json:"name,omitempty"`
	NodeID            string `json:"node_id,omitempty"`
	URL 				      string `json:"url,omitempty"`
	CodespaceURL      string `json:"codespace_url,omitempty"`
}

type Wallet struct {
	NetworkRPCURL string `json:"network_rpc_url,omitempty"`
	NetworkName   string `json:"network_name,omitempty"`
	ChainID       string `json:"chain_id,omitempty"`
	TokenSymbol   string `json:"token_symbol,omitempty"`
	TokenName     string `json:"token_name,omitempty"`
}
