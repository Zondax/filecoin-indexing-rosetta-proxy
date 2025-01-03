package services

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

// ChainIDKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the current chain id
const ChainIDKey = "chainID"

// OptionsSenderIDKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the sender's account ID
const OptionsSenderIDKey = "idSender"

// OptionsReceiverIDKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the receiver's account ID
const OptionsReceiverIDKey = "idReceiver"

// OptionsBlockInclKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse determines on how many epochs message should included
// being 0 the fastest (and the most gas expensive one)
const OptionsBlockInclKey = "blockIncl"

// NonceKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies the next valid nonce.
const NonceKey = "nonce"

// GasPremiumKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies tx's gas premium
const GasPremiumKey = "gasPremium"

// GasLimitKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies tx's gas limit
const GasLimitKey = "gasLimit"

// GasFeeCapKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies gas fee cap
const GasFeeCapKey = "gasFeeCap"

// DestinationActorIdKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies the receiver's actor id
const DestinationActorIdKey = "destinationActorId"

// OptionsValueKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the tokens quantity to be sent
const OptionsValueKey = "value"

// OptionsMethodNumKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the method num
const OptionsMethodNumKey = "methodNum"

// OptionsParamsKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the params
const OptionsParamsKey = "params"

// ConstructionAPIService implements the server.ConstructionAPIServicer interface.
type ConstructionAPIService struct {
	network *types.NetworkIdentifier
	node    api.FullNode
}

var ErrMalformedParams = &types.Error{
	Code:      1000,
	Message:   "malformed params",
	Retriable: false,
}

// NewConstructionAPIService creates a new instance of an ConstructionAPIService.
func NewConstructionAPIService(network *types.NetworkIdentifier, node *api.FullNode) server.ConstructionAPIServicer {
	return &ConstructionAPIService{
		network: network,
		node:    *node,
	}
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (c *ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	var (
		addressSenderParsed   address.Address // nolint
		addressReceiverParsed address.Address // nolint
		err                   error
		nonce                 uint64
		blockInclUint         uint64 = 1
		message                      = &filTypes.Message{
			GasLimit: 0, GasFeeCap: filTypes.NewInt(0),
			GasPremium: filTypes.NewInt(0),
			Value:      abi.NewTokenAmount(1), // Use "1" as default value for better gas estimations
		}
	)

	errNet := rosetta.ValidateNetworkId(ctx, &c.node, request.NetworkIdentifier)
	if errNet != nil {
		return nil, errNet
	}

	md := make(map[string]interface{})

	if request.Options != nil {
		// Parse block include epochs - this field is optional
		blockIncl, ok := request.Options[OptionsBlockInclKey]
		if ok {
			blockInclUint = uint64(blockIncl.(float64))
		}

		// Parse method num - this field is optional
		methodNum, ok := request.Options[OptionsMethodNumKey]
		if ok {
			message.Method = abi.MethodNum(uint64(methodNum.(float64)))
		}

		// Parse params - this field is optional
		params, ok := request.Options[OptionsParamsKey]
		if ok {
			paramsByte, err := base64.StdEncoding.DecodeString(params.(string))
			if err != nil {
				return nil, rosetta.BuildError(ErrMalformedParams, err, false)
			}

			message.Params = paramsByte
		}

		// Parse sender address - this field is optional
		addressSenderRaw, okSender := request.Options[OptionsSenderIDKey]
		if okSender {
			addressSenderParsed, err = address.NewFromString(addressSenderRaw.(string)) // nolint
			if err != nil {
				return nil, rosetta.BuildError(rosetta.ErrInvalidAccountAddress, err, true)
			}
			message.From = addressSenderParsed
		}

		// Parse receiver address - this field is optional
		addressReceiverRaw, okReceiver := request.Options[OptionsReceiverIDKey]
		if okReceiver {
			addressReceiverParsed, err = address.NewFromString(addressReceiverRaw.(string)) // nolint
			if err != nil {
				return nil, rosetta.BuildError(rosetta.ErrInvalidAccountAddress, err, true)
			}
			message.To = addressReceiverParsed

			// Get receiver's actor code
			receiverActor, errAct := c.node.StateGetActor(context.Background(), addressReceiverParsed, filTypes.EmptyTSK)
			if errAct != nil {
				// Actor not found on chain, set an empty field
				md[DestinationActorIdKey] = ""
			} else {
				md[DestinationActorIdKey] = receiverActor.Code.String()
			}
		}

		// Parse value to send - this field is optional
		valueRaw, okValue := request.Options[OptionsValueKey]
		if okValue {
			value, err := filTypes.BigFromString(valueRaw.(string))
			if err != nil {
				return nil, rosetta.BuildError(rosetta.ErrMalformedValue, err, false)
			}
			message.Value = value
		}

		if okSender {
			nonce, err = c.node.MpoolGetNonce(ctx, addressSenderParsed)
			if err != nil {
				return nil, rosetta.BuildError(rosetta.ErrUnableToGetNextNonce, err, true)
			}
			md[NonceKey] = nonce

			// GasEstimateMessageGas
			message, err = c.node.GasEstimateMessageGas(ctx, message, &api.MessageSendSpec{MaxFee: filTypes.NewInt(uint64(build.BlockGasLimit))}, filTypes.TipSetKey{})
			if err != nil {
				return nil, rosetta.BuildError(rosetta.ErrUnableToEstimateGasLimit, err, true)
			}

			// GasEstimateGasPremium
			gasPremium, gasErr := c.node.GasEstimateGasPremium(ctx, blockInclUint, addressSenderParsed, message.GasLimit, filTypes.TipSetKey{}) // nolint
			if gasErr != nil {
				return nil, rosetta.BuildError(rosetta.ErrUnableToEstimateGasPremium, gasErr, true)
			}
			message.GasPremium = gasPremium

			// GasEstimateFeeCap requires gasPremium to be set on message
			gasFeeCap, gasErr := c.node.GasEstimateFeeCap(ctx, message, int64(blockInclUint), filTypes.TipSetKey{})
			if gasErr != nil {
				return nil, rosetta.BuildError(rosetta.ErrUnableToEstimateGasFeeCap, gasErr, true)
			}
			message.GasFeeCap = gasFeeCap
		} else {
			// We can only estimate gas premium without a sender address
			gasPremium, gasErr := c.node.GasEstimateGasPremium(ctx, blockInclUint, address.Address{}, message.GasLimit, filTypes.TipSetKey{}) // nolint
			if gasErr != nil {
				return nil, rosetta.BuildError(rosetta.ErrUnableToEstimateGasPremium, gasErr, true)
			}
			message.GasPremium = gasPremium
		}
	}

	md[GasLimitKey] = message.GasLimit
	md[GasPremiumKey] = message.GasPremium.String()
	md[GasFeeCapKey] = message.GasFeeCap.String()
	md[ChainIDKey] = request.NetworkIdentifier.Network

	resp := &types.ConstructionMetadataResponse{
		Metadata: md,
	}

	return resp, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (c *ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {

	if request.SignedTransaction == "" {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	err := rosetta.ValidateNetworkId(ctx, &c.node, request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}

	rawIn := json.RawMessage(request.SignedTransaction)

	bytes, errJson := rawIn.MarshalJSON()
	if errJson != nil {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	var signedTx filTypes.SignedMessage
	errUnmarshal := json.Unmarshal(bytes, &signedTx)
	if errUnmarshal != nil {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	cid, errTx := c.node.MpoolPush(ctx, &signedTx)
	if errTx != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToSubmitTx, errTx, true)
	}

	resp := &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: cid.String(),
		},
	}

	return resp, nil
}

func (c *ConstructionAPIService) ConstructionCombine(ctx context.Context, request *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}

func (c *ConstructionAPIService) ConstructionDerive(ctx context.Context, request *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}

func (c *ConstructionAPIService) ConstructionHash(ctx context.Context, request *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}

func (c *ConstructionAPIService) ConstructionParse(ctx context.Context, request *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}

func (c *ConstructionAPIService) ConstructionPayloads(ctx context.Context, request *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}

func (c *ConstructionAPIService) ConstructionPreprocess(ctx context.Context, request *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}
