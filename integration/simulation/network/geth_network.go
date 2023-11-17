package network

import (
	"github.com/ten-protocol/go-ten/go/ethadapter"
	"github.com/ten-protocol/go-ten/go/ethadapter/erc20contractlib"
	"github.com/ten-protocol/go-ten/go/ethadapter/mgmtcontractlib"
	"github.com/ten-protocol/go-ten/go/obsclient"
	"github.com/ten-protocol/go-ten/go/rpc"
	"github.com/ten-protocol/go-ten/integration/common/testlog"
	"github.com/ten-protocol/go-ten/integration/eth2network"
	"github.com/ten-protocol/go-ten/integration/simulation/params"
	"github.com/ten-protocol/go-ten/integration/simulation/stats"
)

type networkInMemGeth struct {
	l2Clients []rpc.Client

	// geth
	eth2Network eth2network.Eth2Network
	gethClients []ethadapter.EthClient
	wallets     *params.SimWallets
}

func NewNetworkInMemoryGeth(wallets *params.SimWallets) Network {
	return &networkInMemGeth{
		wallets: wallets,
	}
}

// Create inits and starts the nodes, wires them up, and populates the network objects
func (n *networkInMemGeth) Create(params *params.SimParams, _ *stats.Stats) (*RPCHandles, error) {
	// kickoff the network with the prefunded wallet addresses
	params.L1SetupData, n.gethClients, n.eth2Network = SetUpGethNetwork(
		n.wallets,
		params.StartPort,
		params.NumberOfNodes,
		int(params.AvgBlockDuration.Seconds()),
	)

	params.MgmtContractLib = mgmtcontractlib.NewMgmtContractLib(&params.L1SetupData.MgmtContractAddress, testlog.Logger())
	params.ERC20ContractLib = erc20contractlib.NewERC20ContractLib(&params.L1SetupData.MgmtContractAddress,
		&params.L1SetupData.ObxErc20Address, &params.L1SetupData.EthErc20Address)

	// Start the obscuro nodes and return the handles
	n.l2Clients = startInMemoryObscuroNodes(params, n.eth2Network.GethGenesis(), n.gethClients)

	obscuroClients := make([]*obsclient.ObsClient, params.NumberOfNodes)
	for idx, l2Client := range n.l2Clients {
		obscuroClients[idx] = obsclient.NewObsClient(l2Client)
	}
	walletClients := createAuthClientsPerWallet(n.l2Clients, params.Wallets)

	return &RPCHandles{
		EthClients:     n.gethClients,
		ObscuroClients: obscuroClients,
		RPCClients:     n.l2Clients,
		AuthObsClients: walletClients,
	}, nil
}

func (n *networkInMemGeth) TearDown() {
	// Stop the obscuro nodes first
	StopObscuroNodes(n.l2Clients)

	// Stop geth last
	StopEth2Network(n.gethClients, n.eth2Network)
}
