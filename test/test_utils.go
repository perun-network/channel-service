package test

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"perun.network/channel-service/rpc/proto"
	gpchannel "perun.network/go-perun/channel"
	perunproto "perun.network/go-perun/wire/protobuf"
	ckbasset "perun.network/perun-ckb-backend/channel/asset"
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
	//log.Printf("Requester: %v", requesterPart)
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

func NewChannelUpdateRequest(channelID gpchannel.ID, chState *gpchannel.State, amounts map[gpchannel.Asset]float64, senderId gpchannel.Index) (*proto.ChannelUpdateRequest, error) {
	if channelID != chState.ID {
		log.Println("Channel ID mismatch")
		return nil, errors.New("your error message")
	}

	// 0 for Alice, 1 for Bob
	actorIdx := senderId
	peerIdx := 1 - actorIdx

	for assetType, amount := range amounts {
		if amount < 0 {
			continue
		}
		assetType := assetType.(*ckbasset.Asset)
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
		case *ckbasset.Asset:
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

// ShannonToCKByte converts a given amount in Shannon to CKByte.
func ShannonToCKByte(shannonAmount *big.Int) *big.Float {
	shannonPerCKByte := new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
	shannonPerCKByteFloat := new(big.Float).SetInt(shannonPerCKByte)
	shannonAmountFloat := new(big.Float).SetInt(shannonAmount)
	return new(big.Float).Quo(shannonAmountFloat, shannonPerCKByteFloat)
}

func AllocToString(alloc *gpchannel.Allocation) string {
	var sbArr = make([]strings.Builder, len(alloc.Assets))
	for idx, asset := range alloc.Assets {
		fmt.Fprintf(&sbArr[idx], "Asset type:%v \n", getAssetType(asset))
		fmt.Fprintf(&sbArr[idx], "Asset Allocation: ")
		participant1Balance := ShannonToCKByte(alloc.Balances[idx][0])
		participant2Balance := ShannonToCKByte(alloc.Balances[idx][1])
		fmt.Fprintf(&sbArr[idx], "[%v,%v]", participant1Balance, participant2Balance)
	}
	if len(sbArr) == 0 {
		return "No assets in allocation"
	}

	var sb strings.Builder
	for _, v := range sbArr {
		sb.WriteString(v.String())
	}
	return sb.String()
}

func getAssetType(asset gpchannel.Asset) string {
	assetType := asset.(*ckbasset.Asset)
	if assetType.IsCKBytes {
		return "CKBytes"
	}
	return "SUDT"
}
