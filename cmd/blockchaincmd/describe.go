// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package blockchaincmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ava-labs/avalanche-cli/pkg/cobrautils"
	"github.com/ava-labs/avalanche-cli/pkg/constants"
	"github.com/ava-labs/avalanche-cli/pkg/contract"
	"github.com/ava-labs/avalanche-cli/pkg/key"
	"github.com/ava-labs/avalanche-cli/pkg/localnet"
	"github.com/ava-labs/avalanche-cli/pkg/models"
	"github.com/ava-labs/avalanche-cli/pkg/subnet"
	icmgenesis "github.com/ava-labs/avalanche-cli/pkg/teleporter/genesis"
	"github.com/ava-labs/avalanche-cli/pkg/txutils"
	"github.com/ava-labs/avalanche-cli/pkg/utils"
	"github.com/ava-labs/avalanche-cli/pkg/ux"
	"github.com/ava-labs/avalanche-cli/pkg/vm"
	validatorManagerSDK "github.com/ava-labs/avalanche-cli/sdk/validatormanager"
	anr_utils "github.com/ava-labs/avalanche-network-runner/utils"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/subnet-evm/core"
	"github.com/ava-labs/subnet-evm/params"
	"github.com/ava-labs/subnet-evm/precompile/contracts/deployerallowlist"
	"github.com/ava-labs/subnet-evm/precompile/contracts/feemanager"
	"github.com/ava-labs/subnet-evm/precompile/contracts/nativeminter"
	"github.com/ava-labs/subnet-evm/precompile/contracts/rewardmanager"
	"github.com/ava-labs/subnet-evm/precompile/contracts/txallowlist"
	"github.com/ava-labs/subnet-evm/precompile/contracts/warp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var printGenesisOnly bool
var asJson bool

// avalanche blockchain describe
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [blockchainName]",
		Short: "Print a summary of the blockchain’s configuration",
		Long: `The blockchain describe command prints the details of a Blockchain configuration to the console.
By default, the command prints a summary of the configuration. By providing the --genesis
flag, the command instead prints out the raw genesis file.`,
		RunE: describe,
		Args: cobrautils.ExactArgs(1),
	}
	cmd.Flags().BoolVarP(&printGenesisOnly, "genesis", "g", false, "Print the genesis to the console directly instead of the summary")
	cmd.Flags().BoolVarP(&asJson, "json", "j", false, "Print json serialized information")
	return cmd
}

func printGenesis(blockchainName string) error {
	genesisFile := app.GetGenesisPath(blockchainName)
	gen, err := os.ReadFile(genesisFile)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser(string(gen))
	return nil
}

func GatherSubnetInfo(blockchainName string, onlyLocalnetInfo bool) (models.SubnetInfo, error) {
	subnetInfo := models.SubnetInfo{}

	sc, err := app.LoadSidecar(blockchainName)
	if err != nil {
		return subnetInfo, err
	}

	genesisBytes, err := app.LoadRawGenesis(sc.Subnet)
	if err != nil {
		return subnetInfo, err
	}

	// VM/Deploys
	subnetInfo.Name = sc.Name
	vmIDstr := sc.ImportedVMID
	if vmIDstr == "" {
		if vmID, err := anr_utils.VMID(sc.Name); err == nil {
			vmIDstr = vmID.String()
		} else {
			vmIDstr = constants.NotAvailableLabel
		}
	}

	subnetInfo.VMID = vmIDstr
	subnetInfo.VMVersion = sc.VMVersion
	subnetInfo.Validation = sc.ValidatorManagement

	locallyDeployed, err := localnet.Deployed(sc.Name)
	if err != nil {
		return subnetInfo, err
	}

	var localNetworks []models.LocalNetwork
	index := 0
	localChainID := ""
	blockchainID := ""
	for net, data := range sc.Networks {
		network, err := app.GetNetworkFromSidecarNetworkName(net)
		if err != nil {
			if !asJson {
				ux.Logger.RedXToUser("%s is supposed to be deployed to network %s: %s ", blockchainName, network.Name(), err)
				ux.Logger.PrintToUser("")
			}
			continue
		}
		if network.Kind == models.Local && !locallyDeployed {
			continue
		}
		if network.Kind != models.Local && onlyLocalnetInfo {
			continue
		}

		localNetworks = append(localNetworks, models.LocalNetwork{})
		localNetworks[index].Name = net

		genesisBytes, err := contract.GetBlockchainGenesis(
			app,
			network,
			contract.ChainSpec{
				BlockchainName: sc.Name,
			},
		)
		if err != nil {
			return subnetInfo, err
		}
		if utils.ByteSliceIsSubnetEvmGenesis(genesisBytes) {
			genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisBytes)
			if err != nil {
				return subnetInfo, err
			}

			localNetworks[index].ChainID = (*genesis.Config.ChainID).String()

			if network.Kind == models.Local {
				localChainID = genesis.Config.ChainID.String()
			}
		}
		if data.SubnetID != ids.Empty {

			localNetworks[index].SubnetID = data.SubnetID.String()

			isPermissioned, owners, threshold, err := txutils.GetOwners(network, data.SubnetID)
			if err != nil {
				return subnetInfo, err
			}
			if isPermissioned {

				localNetworks[index].ChainOwners = &models.ChainOwners{Names: owners, Threshold: threshold}

			}
		}
		if data.BlockchainID != ids.Empty {
			blockchainID = data.BlockchainID.String()
			hexEncoding := "0x" + hex.EncodeToString(data.BlockchainID[:])

			localNetworks[index].BlockchainIDCB58 = data.BlockchainID.String()
			localNetworks[index].BlockchainIDHex = hexEncoding

		}
		endpoint, _, err := contract.GetBlockchainEndpoints(
			app,
			network,
			contract.ChainSpec{
				BlockchainName: sc.Name,
			},
			false,
			false,
		)
		if err != nil {
			return subnetInfo, err
		}

		localNetworks[index].RPCEndpoint = endpoint

		// Teleporter
		if data.TeleporterMessengerAddress != "" {

			localNetworks[index].Teleporter.MessengerAddress = data.TeleporterMessengerAddress

		}
		if data.TeleporterRegistryAddress != "" {

			localNetworks[index].Teleporter.RegistryAddress = data.TeleporterRegistryAddress

		}

		index++
	}

	if len(localNetworks) > 0 {
		subnetInfo.LocalNetworks = &localNetworks
	}

	// Token
	subnetInfo.Token = &models.Token{Name: sc.TokenName, Symbol: sc.TokenSymbol}

	if utils.ByteSliceIsSubnetEvmGenesis(genesisBytes) {
		genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisBytes)
		if err != nil {
			return subnetInfo, err
		}
		// Allocation
		allocs, err := gatherAllocations(sc, genesis)
		if err != nil {
			return subnetInfo, err
		}

		subnetInfo.TokenAllocations = &allocs

		// Smart contract
		smartContracts := gatherSmartContracts(sc, genesis)
		subnetInfo.SmartContracts = &smartContracts
		precompiles := gatherPrecompiles(genesis)
		subnetInfo.PrecompileConfigs = &precompiles

	}

	if locallyDeployed {
		ux.Logger.PrintToUser("Local Network Information")
		rpcURLs, nodes, err := localnet.GatherEndpoints(sc.Name)
		if err != nil {
			return subnetInfo, err
		}

		subnetInfo.Nodes = &nodes
		subnetInfo.RPCURLs = &rpcURLs

		localEndpoint := models.NewLocalNetwork().BlockchainEndpoint(blockchainID)
		codespaceEndpoint, err := utils.GetCodespaceURL(localEndpoint)
		if err != nil {
			return subnetInfo, err
		}
		if codespaceEndpoint != "" {
			localEndpoint = codespaceEndpoint
		}

		// Wallet
		subnetInfo.Wallet = &models.Wallet{NetworkName: sc.Name, NetworkRPCURL: localEndpoint, ChainID: localChainID, TokenName: sc.TokenName, TokenSymbol: sc.TokenSymbol}

	}

	return subnetInfo, nil
}

func PrintSubnetInfo(subnetInfo *models.SubnetInfo) {

	// VM/Deploys
	t := table.NewWriter()
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.Style().Options.SeparateRows = true
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true},
	})
	rowConfig := table.RowConfig{AutoMerge: true, AutoMergeAlign: text.AlignLeft}
	t.SetTitle(subnetInfo.Name)
	t.AppendRow(table.Row{"Name", subnetInfo.Name, subnetInfo.Name}, rowConfig)
	t.AppendRow(table.Row{"VM ID", subnetInfo.VMID, subnetInfo.VMID}, rowConfig)
	t.AppendRow(table.Row{"VM Version", subnetInfo.VMVersion, subnetInfo.VMVersion}, rowConfig)
	t.AppendRow(table.Row{"Validation", subnetInfo.Validation, subnetInfo.Validation}, rowConfig)

	if subnetInfo.LocalNetworks != nil {
		for _, localNetwork := range *subnetInfo.LocalNetworks {

			if localNetwork.ChainID != "" {
				t.AppendRow(table.Row{localNetwork.Name, "ChainID", localNetwork.ChainID})
			}
			if localNetwork.SubnetID != "" {
				t.AppendRow(table.Row{localNetwork.Name, "SubnetID", localNetwork.SubnetID})
				if len(localNetwork.ChainOwners.Names) > 0 {
					t.AppendRow(table.Row{localNetwork.Name, fmt.Sprintf("Owners (Threhold=%d)", localNetwork.ChainOwners.Threshold), strings.Join(localNetwork.ChainOwners.Names, "\n")})
				}
			}
			if localNetwork.ChainID != "" {
				t.AppendRow(table.Row{localNetwork.Name, "BlockchainID (CB58)", localNetwork.BlockchainIDCB58})
				t.AppendRow(table.Row{localNetwork.Name, "BlockchainID (HEX)", localNetwork.BlockchainIDHex})
			}
			t.AppendRow(table.Row{localNetwork.Name, "RPC Endpoint", localNetwork.RPCEndpoint})
		}
		ux.Logger.PrintToUser(t.Render())

		// Teleporter
		t = table.NewWriter()
		t.Style().Title.Align = text.AlignCenter
		t.Style().Title.Format = text.FormatUpper
		t.Style().Options.SeparateRows = true
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, AutoMerge: true},
		})
		t.SetTitle("Teleporter")
		hasTeleporterInfo := false
		for _, localNetwork := range *subnetInfo.LocalNetworks {
			if localNetwork.Teleporter.MessengerAddress != "" {
				t.AppendRow(table.Row{localNetwork.Name, "Teleporter Messenger Address", localNetwork.Teleporter.MessengerAddress})
				hasTeleporterInfo = true
			}
			if localNetwork.Teleporter.RegistryAddress != "" {
				t.AppendRow(table.Row{localNetwork.Name, "Teleporter Registry Address", localNetwork.Teleporter.RegistryAddress})
				hasTeleporterInfo = true
			}
		}
		if hasTeleporterInfo {
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser(t.Render())
		}

		// Token
		ux.Logger.PrintToUser("")
		t = table.NewWriter()
		t.Style().Title.Align = text.AlignCenter
		t.Style().Title.Format = text.FormatUpper
		t.Style().Options.SeparateRows = true
		t.SetTitle("Token")
		t.AppendRow(table.Row{"Token Name", subnetInfo.Token.Name})
		t.AppendRow(table.Row{"Token Symbol", subnetInfo.Token.Symbol})
		ux.Logger.PrintToUser(t.Render())
	}

	printAllocations(subnetInfo.TokenAllocations, subnetInfo.Token)
	printSmartContracts(subnetInfo.SmartContracts)
	printPrecompiles(subnetInfo.PrecompileConfigs)

	if subnetInfo.LocalNetworks != nil {
		ux.Logger.PrintToUser("")
		localnet.PrintEndpoints(ux.Logger.PrintToUser, subnetInfo.RPCURLs, subnetInfo.Nodes)

		// Wallet
		t = table.NewWriter()
		t.Style().Title.Align = text.AlignCenter
		t.Style().Title.Format = text.FormatUpper
		t.Style().Options.SeparateRows = true
		t.SetTitle("Wallet Connection")
		t.AppendRow(table.Row{"Network RPC URL", subnetInfo.Wallet.NetworkRPCURL})
		t.AppendRow(table.Row{"Network Name", subnetInfo.Name})
		t.AppendRow(table.Row{"Chain ID", subnetInfo.Wallet.ChainID})
		t.AppendRow(table.Row{"Token Symbol", subnetInfo.Wallet.TokenSymbol})
		t.AppendRow(table.Row{"Token Name", subnetInfo.Wallet.TokenName})
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser(t.Render())
	}
}

func gatherAllocations(sc models.Sidecar, genesis core.Genesis) ([]models.TokenAlloc, error) {
	tokenAllocations := []models.TokenAlloc{}
	teleporterKeyAddress := ""
	if sc.TeleporterReady {
		k, err := key.LoadSoft(models.NewLocalNetwork().ID, app.GetKeyPath(sc.TeleporterKey))
		if err != nil {
			return tokenAllocations, err
		}
		teleporterKeyAddress = k.C()
	}
	_, subnetAirdropAddress, _, err := subnet.GetDefaultSubnetAirdropKeyInfo(app, sc.Name)
	if err != nil {
		return tokenAllocations, err
	}
	if len(genesis.Alloc) > 0 {
		for address, allocation := range genesis.Alloc {
			amount := allocation.Balance
			// we are only interested in supply distribution here
			if amount == nil || big.NewInt(0).Cmp(amount) == 0 {
				continue
			}
			formattedAmount := new(big.Int).Div(amount, big.NewInt(params.Ether))
			description := ""
			privKey := ""
			switch address.Hex() {
			case teleporterKeyAddress:
				description = "Used by ICM"
			case subnetAirdropAddress:
				description = "Main funded account"
			case vm.PrefundedEwoqAddress.Hex():
				description = "Main funded account"
			case sc.ValidatorManagerOwner:
				description = "Validator Manager Owner"
			case sc.ProxyContractOwner:
				description = "Proxy Admin Owner"
			}
			var (
				found bool
				name  string
			)
			found, name, _, privKey, err = contract.SearchForManagedKey(app, models.NewLocalNetwork(), address, true)
			if err != nil {
				return tokenAllocations, err
			}
			tokenAllocations = append(tokenAllocations, models.TokenAlloc{Description: description, Address: address.Hex(), PrivateKey: privKey, AmountToken: formattedAmount.String(), AmountWEI: amount.String()})
			if found {
				tokenAllocations[len(tokenAllocations)-1].Name = name
			}
		}
	}
	return tokenAllocations, nil
}

func printAllocations(tokenAllocations *[]models.TokenAlloc, token *models.Token) {
	if tokenAllocations == nil && len(*tokenAllocations) == 0 {
		return 
	}
		ux.Logger.PrintToUser("")
		t := table.NewWriter()
		t.Style().Title.Align = text.AlignCenter
		t.Style().Title.Format = text.FormatUpper
		t.Style().Options.SeparateRows = true
		t.SetTitle("Initial Token Allocation")
		t.AppendHeader(table.Row{
			"Description",
			"Address and Private Key",
			fmt.Sprintf("Amount (%s)", token.Symbol),
			"Amount (wei)",
		})
		for _, allocation := range *tokenAllocations {
			t.AppendRow(table.Row{fmt.Sprintf("%s\n%s", logging.Orange.Wrap(allocation.Description), allocation.Name), allocation.Address + "\n" + allocation.PrivateKey, allocation.AmountToken, allocation.AmountWEI})
		}
		ux.Logger.PrintToUser(t.Render())
}

func gatherSmartContracts(sc models.Sidecar, genesis core.Genesis) []models.SmartContract {
	smartContracts := []models.SmartContract{}
	if len(genesis.Alloc) == 0 {
		return smartContracts
	}
	for address, allocation := range genesis.Alloc {
		if len(allocation.Code) == 0 {
			continue
		}
		var description, deployer string
		switch {
		case address == common.HexToAddress(icmgenesis.MessengerContractAddress):
			description = "ICM Messenger"
			deployer = icmgenesis.MessengerDeployerAddress
		case address == common.HexToAddress(validatorManagerSDK.ValidatorContractAddress):
			if sc.PoA() {
				description = "PoA Validator Manager"
			} else {
				description = "Native Token Staking Manager"
			}
		case address == common.HexToAddress(validatorManagerSDK.ProxyContractAddress):
			description = "Transparent Proxy"
		case address == common.HexToAddress(validatorManagerSDK.ProxyAdminContractAddress):
			description = "Proxy Admin"
			deployer = sc.ProxyContractOwner
		case address == common.HexToAddress(validatorManagerSDK.RewardCalculatorAddress):
			description = "Reward Calculator"
		}
		smartContracts = append(smartContracts, models.SmartContract{Description: description, Address: address.Hex(), Deployer: deployer})
	}
	return smartContracts
}

func printSmartContracts(smartContracts *[]models.SmartContract) {
	if smartContracts == nil && len(*smartContracts) == 0 {
		return
	}
	ux.Logger.PrintToUser("")
	t := table.NewWriter()
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.Style().Options.SeparateRows = true
	t.SetTitle("Smart Contracts")
	t.AppendHeader(table.Row{"Description", "Address", "Deployer"})
	for _, smartContract := range *smartContracts {

		t.AppendRow(table.Row{smartContract.Description, smartContract.Address, smartContract.Deployer})
	}
	ux.Logger.PrintToUser(t.Render())
}

func gatherPrecompiles(genesis core.Genesis) []models.Precompile {
	precompiles := []models.Precompile{}
	// Warp
	if genesis.Config.GenesisPrecompiles[warp.ConfigKey] != nil {
		precompiles = append(precompiles, models.Precompile{Name: "Warp"})
	}
	// Native Minting
	if genesis.Config.GenesisPrecompiles[nativeminter.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[nativeminter.ConfigKey].(*nativeminter.Config)
		precompiles = append(precompiles, models.Precompile{Name: "Native Minter", AdminAddresses: cfg.AdminAddresses, ManagerAddresses: cfg.ManagerAddresses, EnabledAddresses: cfg.EnabledAddresses})
	}
	// Contract allow list
	if genesis.Config.GenesisPrecompiles[deployerallowlist.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[deployerallowlist.ConfigKey].(*deployerallowlist.Config)
		precompiles = append(precompiles, models.Precompile{Name: "Contract Allow List", AdminAddresses: cfg.AdminAddresses, ManagerAddresses: cfg.ManagerAddresses, EnabledAddresses: cfg.EnabledAddresses})
	}
	// TX allow list
	if genesis.Config.GenesisPrecompiles[txallowlist.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[txallowlist.Module.ConfigKey].(*txallowlist.Config)
		precompiles = append(precompiles, models.Precompile{Name: "Tx Allow List", AdminAddresses: cfg.AdminAddresses, ManagerAddresses: cfg.ManagerAddresses, EnabledAddresses: cfg.EnabledAddresses})
	}
	// Fee config allow list
	if genesis.Config.GenesisPrecompiles[feemanager.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[feemanager.ConfigKey].(*feemanager.Config)
		precompiles = append(precompiles, models.Precompile{Name: "Fee Config Allow List", AdminAddresses: cfg.AdminAddresses, ManagerAddresses: cfg.ManagerAddresses, EnabledAddresses: cfg.EnabledAddresses})
	}
	// Reward config allow list
	if genesis.Config.GenesisPrecompiles[rewardmanager.ConfigKey] != nil {
		cfg := genesis.Config.GenesisPrecompiles[rewardmanager.ConfigKey].(*rewardmanager.Config)
		precompiles = append(precompiles, models.Precompile{Name: "Reward Manager Allow List", AdminAddresses: cfg.AdminAddresses, ManagerAddresses: cfg.ManagerAddresses, EnabledAddresses: cfg.EnabledAddresses})
	}
	return precompiles
}

func printPrecompiles(precompiles *[]models.Precompile) {
	if precompiles == nil && len(*precompiles) == 0 {
		return
	}
	ux.Logger.PrintToUser("")
	t := table.NewWriter()
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true},
	})
	t.SetTitle("Initial Precompile Configs")
	t.AppendHeader(table.Row{"Precompile", "Admin Addresses", "Manager Addresses", "Enabled Addresses"})

	warpSet := false
	allowListSet := false
	for _, precompile := range *precompiles {
		if precompile.Name == "Warp" {
			warpSet = true
			t.AppendRow(table.Row{"Warp", "n/a", "n/a", "n/a"})
		} else {
			allowListSet = true
			addPrecompileAllowListToTable(t, precompile.Name, precompile.AdminAddresses, precompile.ManagerAddresses, precompile.EnabledAddresses)
		}
	}
	if warpSet || allowListSet {
		ux.Logger.PrintToUser(t.Render())
		if allowListSet {
			note := logging.Orange.Wrap("The allowlist is taken from the genesis and is not being updated if you make adjustments\nvia the precompile. Use readAllowList(address) instead.")
			ux.Logger.PrintToUser(note)
		}
	}
}

func addPrecompileAllowListToTable(
	t table.Writer,
	label string,
	adminAddresses []common.Address,
	managerAddresses []common.Address,
	enabledAddresses []common.Address,
) {
	t.AppendSeparator()
	admins := len(adminAddresses)
	managers := len(managerAddresses)
	enabled := len(enabledAddresses)
	max := max(admins, managers, enabled)
	for i := 0; i < max; i++ {
		var admin, manager, enable string
		if i < len(adminAddresses) && adminAddresses[i] != (common.Address{}) {
			admin = adminAddresses[i].Hex()
		}
		if i < len(managerAddresses) && managerAddresses[i] != (common.Address{}) {
			manager = managerAddresses[i].Hex()
		}
		if i < len(enabledAddresses) && enabledAddresses[i] != (common.Address{}) {
			enable = enabledAddresses[i].Hex()
		}
		t.AppendRow(table.Row{label, admin, manager, enable})
	}
}

func describe(_ *cobra.Command, args []string) error {
	blockchainName := args[0]
	if !app.GenesisExists(blockchainName) {
		ux.Logger.PrintToUser("The provided subnet name %q does not exist", blockchainName)
		return nil
	}
	if printGenesisOnly {
		return printGenesis(blockchainName)
	}
	subnetInfo, err := GatherSubnetInfo(blockchainName, false) // TODO: Show all gathering information before returning the error
	if err != nil {
		return err
	}
	if !asJson {
		PrintSubnetInfo(&subnetInfo)
		if isEVM, _, err := app.HasSubnetEVMGenesis(blockchainName); err != nil {
			return err
		} else if !isEVM {
			sc, err := app.LoadSidecar(blockchainName)
			if err != nil {
				return err
			}
			app.Log.Warn("Unknown genesis format", zap.Any("vm-type", sc.VM))
			ux.Logger.PrintToUser("")
			ux.Logger.PrintToUser("Printing genesis")
			return printGenesis(blockchainName)
		}
	} else if err = ux.Logger.PrintJSONToUser(subnetInfo); err != nil {
		return err
	}
	return nil
}
