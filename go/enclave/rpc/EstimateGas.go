package rpc

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/ten-protocol/go-ten/go/common"
	"github.com/ten-protocol/go-ten/go/common/gethapi"
	"github.com/ten-protocol/go-ten/go/common/gethencoding"
	"github.com/ten-protocol/go-ten/go/common/measure"
	"github.com/ten-protocol/go-ten/go/common/syserr"
	"github.com/ten-protocol/go-ten/go/enclave"
	"github.com/ten-protocol/go-ten/go/enclave/core"
)

type EstimateGasParams struct {
	TransactionArgs *gethapi.TransactionArgs
	BlockNumber     *gethrpc.BlockNumber
}

func ExtractEstimateGasRequest(reqParams []any) (*enclave.UserRPCRequest[EstimateGasParams], error) {
	// Parameters are [callMsg, block number (optional)]
	if len(reqParams) < 1 {
		return nil, fmt.Errorf("unexpected number of parameters")
	}

	callMsg, err := gethencoding.ExtractEthCall(reqParams[0])
	if err != nil {
		return nil, fmt.Errorf("unable to decode EthCall Params - %w", err)
	}

	// encryption will fail if From address is not provided
	if callMsg.From == nil {
		return nil, fmt.Errorf("no from address provided")
	}

	// extract optional block number - defaults to the latest block if not avail
	blockNumber, err := gethencoding.ExtractOptionalBlockNumber(reqParams, 1)
	if err != nil {
		return nil, fmt.Errorf("unable to extract requested block number - %w", err)
	}

	return &enclave.UserRPCRequest[EstimateGasParams]{
		Sender: callMsg.From,
		Params: &EstimateGasParams{callMsg, blockNumber},
	}, nil
}

func ExecuteEstimateGas(rpc *EncryptionManager, req *enclave.UserRPCRequest[EstimateGasParams]) (*enclave.UserResponse[hexutil.Uint64], error) {
	block, err := rpc.blockResolver.FetchHeadBlock()
	if err != nil {
		return nil, err
	}

	// The message is run through the l1 publishing cost estimation for the current
	// known head block.
	l1Cost, err := rpc.gasOracle.EstimateL1CostForMsg(req.Params.TransactionArgs, block)
	if err != nil {
		return nil, err
	}

	batch, err := rpc.storage.FetchHeadBatch()
	if err != nil {
		return nil, err
	}

	// We divide the total estimated l1 cost by the l2 fee per gas in order to convert
	// the expected cost into l2 gas based on current pricing.
	// todo @siliev - add overhead when the base fee becomes dynamic.
	publishingGas := big.NewInt(0).Div(l1Cost, batch.Header.BaseFee)

	// The one additional gas captures the modulo leftover in some edge cases
	// where BaseFee is bigger than the l1cost.
	publishingGas = big.NewInt(0).Add(publishingGas, gethcommon.Big1)

	executionGasEstimate, err := rpc.doEstimateGas(req.Params.TransactionArgs, req.Params.BlockNumber, rpc.config.GasLocalExecutionCapFlag)
	if err != nil {
		err = fmt.Errorf("unable to estimate transaction - %w", err)

		// make sure it's not some internal error
		if errors.Is(err, syserr.InternalError{}) {
			return nil, err
		}

		// make sure to serialize any possible EVM error
		evmErr, err := serializeEVMError(err)
		if err == nil {
			err = fmt.Errorf(string(evmErr))
		}
		return &enclave.UserResponse[hexutil.Uint64]{nil, err}, nil
	}

	totalGasEstimate := hexutil.Uint64(publishingGas.Uint64() + uint64(executionGasEstimate))

	return &enclave.UserResponse[hexutil.Uint64]{&totalGasEstimate, nil}, nil
}

// DoEstimateGas returns the estimation of minimum gas required to execute transaction
// This is a copy of https://github.com/ethereum/go-ethereum/blob/master/internal/ethapi/api.go#L1055
// there's a high complexity to the method due to geth business rules (which is mimic'd here)
// once the work of obscuro gas mechanics is established this method should be simplified
func (rpc *EncryptionManager) doEstimateGas(args *gethapi.TransactionArgs, blkNumber *gethrpc.BlockNumber, gasCap uint64) (hexutil.Uint64, common.SystemError) { //nolint: gocognit
	// Binary search the gas requirement, as it may be higher than the amount used
	var ( //nolint: revive
		lo  = params.TxGas - 1
		hi  uint64
		cap uint64 //nolint:predeclared
	)
	// Use zero address if sender unspecified.
	if args.From == nil {
		args.From = new(gethcommon.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		// todo (#627) - review this with the gas mechanics/tokenomics work
		/*
			//Retrieve the block to act as the gas ceiling
			block, Err := b.BlockByNumberOrHash(ctx, blockNrOrHash)
			if Err != nil {
				return 0, Err
			}
			if block == nil {
				return 0, errors.New("block not found")
			}
			hi = block.GasLimit()
		*/
		hi = rpc.config.GasLocalExecutionCapFlag
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = gethcommon.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 { //nolint:nestif
		balance, err := rpc.chain.GetBalanceAtBlock(*args.From, blkNumber)
		if err != nil {
			return 0, fmt.Errorf("unable to fetch account balance - %w", err)
		}

		available := new(big.Int).Set(balance.ToInt())
		if args.Value != nil {
			if args.Value.ToInt().Cmp(available) >= 0 {
				return 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			rpc.logger.Debug("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}
	// Recap the highest gas allowance with specified gascap.
	if gasCap != 0 && hi > gasCap {
		rpc.logger.Debug("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}
	cap = hi //nolint: revive

	// Execute the binary search and hone in on an isGasEnough gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := rpc.isGasEnough(args, mid, blkNumber)
		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap { //nolint:nestif
		failed, result, err := rpc.isGasEnough(args, hi, blkNumber)
		if err != nil {
			return 0, err
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas { //nolint: errorlint
				if len(result.Revert()) > 0 {
					return 0, newRevertError(result)
				}
				return 0, result.Err
			}
			// Otherwise, the specified gas cap is too low
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)
		}
	}
	return hexutil.Uint64(hi), nil
}

// Create a helper to check if a gas allowance results in an executable transaction
// isGasEnough returns whether the gaslimit should be raised, lowered, or if it was impossible to execute the message
func (rpc *EncryptionManager) isGasEnough(args *gethapi.TransactionArgs, gas uint64, blkNumber *gethrpc.BlockNumber) (bool, *gethcore.ExecutionResult, error) {
	defer core.LogMethodDuration(rpc.logger, measure.NewStopwatch(), "enclave.go:IsGasEnough")
	args.Gas = (*hexutil.Uint64)(&gas)
	result, err := rpc.chain.ObsCallAtBlock(args, blkNumber)
	if err != nil {
		if errors.Is(err, gethcore.ErrIntrinsicGas) {
			return true, nil, nil // Special case, raise gas limit
		}
		return true, nil, err // Bail out
	}
	return result.Failed(), result, nil
}

func newRevertError(result *gethcore.ExecutionResult) *revertError {
	reason, errUnpack := abi.UnpackRevert(result.Revert())
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(result.Revert()),
	}
}

// revertError is an API error that encompasses an EVM revertal with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// ErrorCode returns the JSON error code for a revertal.
// See: https://github.com/ethereum/wiki/wiki/JSON-RPC-Error-Codes-Improvement-Proposal
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}
