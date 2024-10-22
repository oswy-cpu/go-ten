package rpc

import (
	"fmt"

	"github.com/ten-protocol/go-ten/go/common/privacy"
	enclaveconfig "github.com/ten-protocol/go-ten/go/enclave/config"
	"github.com/ten-protocol/go-ten/go/enclave/gas"

	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ten-protocol/go-ten/go/enclave/components"
	"github.com/ten-protocol/go-ten/go/enclave/crosschain"
	"github.com/ten-protocol/go-ten/go/enclave/l2chain"
	"github.com/ten-protocol/go-ten/go/enclave/nodetype"
	"github.com/ten-protocol/go-ten/go/enclave/storage"

	"github.com/ethereum/go-ethereum/crypto/ecies"
)

// EncryptionManager manages the decryption and encryption of enclave comms.
type EncryptionManager struct {
	chain                  l2chain.ObscuroChain
	enclavePrivateKeyECIES *ecies.PrivateKey
	storage                storage.Storage
	registry               components.BatchRegistry
	processors             *crosschain.Processors
	service                nodetype.NodeType
	gasOracle              gas.Oracle
	blockResolver          storage.BlockResolver
	l1BlockProcessor       components.L1BlockProcessor
	config                 *enclaveconfig.EnclaveConfig
	logger                 gethlog.Logger
	whitelist              *privacy.Whitelist
}

func NewEncryptionManager(enclavePrivateKeyECIES *ecies.PrivateKey, storage storage.Storage, registry components.BatchRegistry, processors *crosschain.Processors, service nodetype.NodeType, config *enclaveconfig.EnclaveConfig, oracle gas.Oracle, blockResolver storage.BlockResolver, l1BlockProcessor components.L1BlockProcessor, chain l2chain.ObscuroChain, logger gethlog.Logger) *EncryptionManager {
	return &EncryptionManager{
		storage:                storage,
		registry:               registry,
		processors:             processors,
		service:                service,
		chain:                  chain,
		config:                 config,
		blockResolver:          blockResolver,
		l1BlockProcessor:       l1BlockProcessor,
		gasOracle:              oracle,
		logger:                 logger,
		enclavePrivateKeyECIES: enclavePrivateKeyECIES,
		whitelist:              privacy.NewWhitelist(),
	}
}

// DecryptBytes decrypts the bytes with the enclave's private key.
func (rpc *EncryptionManager) DecryptBytes(encryptedBytes []byte) ([]byte, error) {
	bytes, err := rpc.enclavePrivateKeyECIES.Decrypt(encryptedBytes, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt bytes with enclave private key. Cause: %w", err)
	}

	return bytes, nil
}
