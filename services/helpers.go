package services

import (
	"context"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/lotus/api"
)

func ValidateNetworkId(ctx context.Context, node *api.FullNode, networkId *types.NetworkIdentifier) *types.Error {

	if networkId == nil {
		return ErrMalformedValue
	}

	fullAPI := *node
	validNetwork, err := fullAPI.StateNetworkName(ctx)
	if err != nil {
		return BuildError(ErrUnableToRetrieveNetworkName, err, true)
	}

	if networkId.Network != string(validNetwork) {
		return BuildError(ErrInvalidNetwork, nil, true)
	}

	return nil
}
