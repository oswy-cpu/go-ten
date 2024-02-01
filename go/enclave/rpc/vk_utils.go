package rpc

import (
	"encoding/json"
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ten-protocol/go-ten/go/common"
	"github.com/ten-protocol/go-ten/go/enclave/vkhandler"
	"github.com/ten-protocol/go-ten/go/responses"
)

// UserRPCRequest1 - contains the values
type UserRPCRequest1[P any] struct {
	Sender *gethcommon.Address
	Param1 *P
}

type UserRPCRequest2[P any, Q any] struct {
	Sender *gethcommon.Address
	Param1 *P
	Param2 *Q
}

// handles the VK management, authentication and encryption
// P represents the single request parameter
func withVKEncryption1[P any](
	encManager *EncryptionManager,
	chainID int64,
	encReq []byte, // encrypted request that contains a signed viewing key
	extractFromAndParams func([]any) (*UserRPCRequest1[P], error), // extract the arguments and the logical sender from the plaintext request. Make sure to not return any information from the db in the error.
	execute func(*UserRPCRequest1[P]) (any, error, error), // execute the user call. Returns a user error or a system error
) (*responses.EnclaveResponse, common.SystemError) {
	return withVKEncryption2[P, P](encManager,
		chainID,
		encReq,
		func(params []any) (*UserRPCRequest2[P, P], error) {
			res, err := extractFromAndParams(params)
			if err != nil {
				return nil, err
			}
			if res == nil {
				return nil, nil
			}
			return &UserRPCRequest2[P, P]{res.Sender, res.Param1, nil}, nil
		},
		func(req *UserRPCRequest2[P, P]) (any, error, error) {
			return execute(&UserRPCRequest1[P]{req.Sender, req.Param1})
		})
}

func withVKEncryption2[P any, Q any](
	encManager *EncryptionManager,
	chainID int64,
	encReq []byte, // encrypted request that contains a signed viewing key
	extractFromAndParams func([]any) (*UserRPCRequest2[P, Q], error), // extract the arguments and the logical sender from the plaintext request. Make sure to not return any information from the db in the error.
	execute func(*UserRPCRequest2[P, Q]) (any, error, error), // execute the user call. Returns a user error or a system error
) (*responses.EnclaveResponse, common.SystemError) {
	// 1. Decrypt request
	plaintextRequest, err := encManager.DecryptBytes(encReq)
	if err != nil {
		return responses.AsPlaintextError(fmt.Errorf("could not decrypt params - %w", err)), nil
	}

	// 2. Unmarshall into a generic []any array
	var decodedRequest []any
	if err := json.Unmarshal(plaintextRequest, &decodedRequest); err != nil {
		return responses.AsPlaintextError(fmt.Errorf("could not unmarshal params - %w", err)), nil
	}

	// 3. Extract the VK from the first element
	if len(decodedRequest) < 1 {
		return responses.AsPlaintextError(fmt.Errorf("invalid request. viewing key is missing")), nil
	}
	vk, err := vkhandler.ExtractAndAuthenticateViewingKey(decodedRequest[0], chainID)
	if err != nil {
		return responses.AsPlaintextError(fmt.Errorf("invalid viewing key - %w", err)), nil
	}

	// 4. Call the function that knows how to extract request specific params from the request
	decodedParams, err := extractFromAndParams(decodedRequest[1:])
	if err != nil {
		return responses.AsEncryptedError(fmt.Errorf("unable to decode params - %w", err), vk), nil
	}

	// when all return values are null, by convention this is "Not found", so we just return an empty value
	if decodedParams == nil && err == nil {
		// todo - this must be encrypted
		// return responses.AsEncryptedEmptyResponse(vk), nil
		return responses.AsEmptyResponse(), nil
	}

	// 5. Validate the logical sender
	if decodedParams.Sender == nil {
		return responses.AsEncryptedError(fmt.Errorf("invalid request - `from` field is mandatory"), vk), nil
	}
	if decodedParams.Sender.Hex() != vk.AccountAddress.Hex() {
		return responses.AsEncryptedError(fmt.Errorf("viewing key account address: %s -does not match the requester: %s", vk.AccountAddress, decodedParams.Sender), vk), nil
	}

	// 6. Make the backend call and convert the response.
	result, userErr, sysErr := execute(decodedParams)
	if sysErr != nil {
		return nil, responses.ToInternalError(sysErr)
	}
	if userErr != nil {
		return responses.AsEncryptedError(userErr, vk), nil
	}
	if result == nil {
		return responses.AsEncryptedEmptyResponse(vk), nil
	}
	return responses.AsEncryptedResponse[any](&result, vk), nil
}
