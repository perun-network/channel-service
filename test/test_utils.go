package test

import (
	"errors"
	"fmt"
	"log"
	"math/big"

	"perun.network/channel-service/rpc/proto"
	gpchannel "perun.network/go-perun/channel"
	perunproto "perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/wallet/address"
)

func NewChannelOpenRequest(requester address.Participant, peer address.Participant, amounts map[gpchannel.Asset]float64) (proto.ChannelOpenRequest, error) {
	challengeDuration := 10
	alloc := newAllocation(amounts)
	protAlloc, err := perunproto.FromAllocation(*alloc)
	if err != nil {
		return proto.ChannelOpenRequest{}, fmt.Errorf("failed to convert allocation: %w", err)
	}
	requesterPart, err := requester.PackOffChainParticipant()
	if err != nil {
		return proto.ChannelOpenRequest{}, fmt.Errorf("failed to pack requester participant: %w", err)
	}
	log.Printf("Requester: %v", requesterPart)
	requesterCkbAddrInBytes := requesterPart.AsSlice()
	peerPart, err := peer.PackOffChainParticipant()
	if err != nil {
		return proto.ChannelOpenRequest{}, fmt.Errorf("failed to pack peer participant: %w", err)
	}
	peerCkbAddrInBytes := peerPart.AsSlice()
	return proto.ChannelOpenRequest{
		Requester:         requesterCkbAddrInBytes,
		Peer:              peerCkbAddrInBytes,
		Allocation:        protAlloc,
		ChallengeDuration: uint64(challengeDuration),
	}, nil
}

func NewPerunClientRequest() *proto.NewPerunClientRequest {
	return &proto.NewPerunClientRequest{}
}

func GetChannelsRequest(requestingParty []byte) *proto.GetChannelsRequest {
	return &proto.GetChannelsRequest{
		Requester: requestingParty,
	}
}

func NewChannelUpdateRequest(channelID gpchannel.ID, chState *gpchannel.State, amounts map[gpchannel.Asset]float64) (*proto.ChannelUpdateRequest, error) {
	// Channel update request contains a protobuf.State which is the protbuf message type for channel state
	// thus I need to create a channel state with the updated balances, convert it back to protobuf and then create the ChannelUpdateRequest
	// to create a new channel state with updated balances, I can take the old state of the participant who's proposing the new update
	// and then change the allocation to the new allocation
	// thus I need access to the user struct of the participant
	if channelID != chState.ID {
		log.Println("Channel ID mismatch")
		return nil, errors.New("your error message")
	}

	// 0 for Alice, 1 for Bob
	actorIdx := gpchannel.Index(0)
	peerIdx := 1 - actorIdx

	for assetType, amount := range amounts {
		if amount < 0 {
			continue
		}
		assetType := assetType.(*asset.Asset)
		if assetType.IsCKBytes {
			shannonAmount := cKByteToShannon(big.NewFloat(amount))
			chState.Allocation.TransferBalance(actorIdx, peerIdx, assetType, shannonAmount)
		}
	}

	protoState, err := perunproto.FromState(chState)
	if err != nil {
		log.Fatalf("Error: Cannot convert channel state to protobuf state")
		return nil, err
	}
	return &proto.ChannelUpdateRequest{
		State: protoState,
	}, nil

}

func newAllocation(amounts map[gpchannel.Asset]float64) *gpchannel.Allocation {
	assets := make([]gpchannel.Asset, len(amounts))
	i := 0
	for a := range amounts {
		assets[i] = a
		i++
	}
	// We create an initial allocation which defines the starting balances.
	initAlloc := gpchannel.NewAllocation(2, assets...)
	log.Println(initAlloc.Assets)
	for a, amount := range amounts {
		switch a := a.(type) {
		case *asset.Asset:
			if a.IsCKBytes {
				initAlloc.SetAssetBalances(a, []gpchannel.Bal{
					cKByteToShannon(big.NewFloat(amount)), // Our initial balance.
					cKByteToShannon(big.NewFloat(amount)), // Peer's initial balance.
				})
			} else {
				intAmount := new(big.Int).SetUint64(uint64(amount))
				initAlloc.SetAssetBalances(a, []gpchannel.Bal{
					intAmount, // Our initial balance.
					intAmount, // Peer's initial balance.
				})
			}
		default:
			panic("Asset is not of type *asset.Asset")
		}

	}
	log.Println("Created Allocation")
	return initAlloc
}

// CKByteToShannon converts a given amount in CKByte to Shannon.
func cKByteToShannon(ckbyteAmount *big.Float) (shannonAmount *big.Int) {
	shannonPerCKByte := new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
	shannonPerCKByteFloat := new(big.Float).SetInt(shannonPerCKByte)
	shannonAmountFloat := new(big.Float).Mul(ckbyteAmount, shannonPerCKByteFloat)
	shannonAmount, _ = shannonAmountFloat.Int(nil)
	return shannonAmount
}
