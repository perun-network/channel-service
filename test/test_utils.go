package test

import (
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

	/*
		requesterCkbAddr, err := requester.ToCKBAddress(types.NetworkTest).EncodeFullBech32m()
		if err != nil {
			return proto.ChannelOpenRequest{}, fmt.Errorf("failed to encode requester address: %w", err)
		}
		requesterCkbAddrInBytes := []byte(requesterCkbAddr)
		peerCkbAddr, err := peer.ToCKBAddress(types.NetworkTest).EncodeFullBech32m()
		if err != nil {
			return proto.ChannelOpenRequest{}, fmt.Errorf("failed to encode peer address: %w", err)
		}
		peerCkbAddrInBytes := []byte(peerCkbAddr)
	*/
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
