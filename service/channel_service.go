package service

import (
	"context"
	"errors"
	"fmt"
	address2 "github.com/nervosnetwork/ckb-sdk-go/v2/address"
	"github.com/nervosnetwork/ckb-sdk-go/v2/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/wallet"
	"perun.network/go-perun/channel"
	gpwallet "perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/channel/adjudicator"
	"perun.network/perun-ckb-backend/channel/funder"
	"perun.network/perun-ckb-backend/client"
	"perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/wallet/external"
)

type ChannelService struct {
	UserRegister UserRegister
	wsc          proto.WalletServiceClient
	bus          wire.Bus
	network      types.Network
	node         rpc.Client
	deployment   backend.Deployment
	wallet       gpwallet.Wallet

	proto.UnimplementedChannelServiceServer // always embed
}

func NewChannelService(c proto.WalletServiceClient, bus wire.Bus, network types.Network, nodeUrl string, deployment backend.Deployment) (*ChannelService, error) {
	node, err := rpc.Dial(nodeUrl)
	if err != nil {
		return nil, err
	}

	return &ChannelService{
		UserRegister: NewMutexUserRegister(),
		wsc:          c,
		bus:          bus,
		network:      network,
		node:         node,
		deployment:   deployment,
		wallet:       external.NewWallet(wallet.NewExternalClient(c)),
	}, nil
}

func (c ChannelService) OpenChannel(ctx context.Context, request *proto.ChannelOpenRequest) (*proto.ChannelOpenResponse, error) {

	user, err := c.GetUserFromChannelOpenRequest(request)
	if err != nil {
		return nil, err
	}
	allocation, err := c.GetAllocationFromChannelOpenRequest(request)
	if err != nil {
		return nil, err
	}
	peer := c.GetPeerAddressFromChannelOpenRequest(request)
	challengeDuration := c.GetChallengeDurationFromChannelOpenRequest(request)
	id, err := user.OpenChannel(ctx, peer, allocation, challengeDuration)
	if err != nil {
		return &proto.ChannelOpenResponse{Msg: &proto.ChannelOpenResponse_Rejected{Rejected: &proto.Rejected{Reason: err.Error()}}}, err
	}
	return &proto.ChannelOpenResponse{Msg: &proto.ChannelOpenResponse_ChannelId{ChannelId: id[:]}}, nil

}

func (c ChannelService) UpdateChannel(ctx context.Context, request *proto.ChannelUpdateRequest) (*proto.ChannelUpdateResponse, error) {
	cid, user, err := c.GetChannelInfoFromRequest(request.State.GetId())
	if err != nil {
		return nil, err
	}
	state, err := AsChannelState(request.GetState())
	if err != nil {
		return nil, err
	}
	err = user.UpdateChannel(ctx, cid, state)
	if err != nil {
		rejected := proto.Rejected{Reason: err.Error()}
		return &proto.ChannelUpdateResponse{Msg: &proto.ChannelUpdateResponse_Rejected{Rejected: &rejected}}, err
	}

	return &proto.ChannelUpdateResponse{Msg: &proto.ChannelUpdateResponse_Update{Update: &proto.SuccessfulUpdate{
		// TODO: Use actual resulting state instead of the request state.
		State:     request.State,
		ChannelId: cid[:],
	}}}, nil
}

func (c ChannelService) CloseChannel(ctx context.Context, request *proto.ChannelCloseRequest) (*proto.ChannelCloseResponse, error) {
	cid, user, err := c.GetChannelInfoFromRequest(request.GetChannelId())
	if err != nil {
		return nil, err
	}
	err = user.CloseChannel(ctx, cid)
	if err != nil {
		// TODO: Do we want to return the error here?
		return &proto.ChannelCloseResponse{Msg: &proto.ChannelCloseResponse_Rejected{Rejected: &proto.Rejected{Reason: err.Error()}}}, err
	}
	return &proto.ChannelCloseResponse{Msg: &proto.ChannelCloseResponse_Close{Close: &proto.SuccessfulClose{ChannelId: cid[:]}}}, nil
}

func AsChannelID(in []byte) (channel.ID, error) {
	id := channel.ID{}
	n := copy(id[:], in)
	if n != len(id) {
		return channel.ID{}, fmt.Errorf("channel id too short: expected %d bytes, got %d", len(id), n)
	}
	return id, nil
}

func AsChannelState(ps *protobuf.State) (*channel.State, error) {
	return protobuf.ToState(ps)
}

func (c ChannelService) GetChannelInfoFromRequest(reqChannelId []byte) (channel.ID, *User, error) {
	cid, err := AsChannelID(reqChannelId)
	if err != nil {
		return channel.ID{}, nil, err
	}
	usr, err := c.UserRegister.GetUser(cid)
	return cid, usr, err
}

func (c ChannelService) GetUserFromChannelOpenRequest(request *proto.ChannelOpenRequest) (*User, error) {
	requester := request.GetRequester()
	if requester == nil {
		return nil, fmt.Errorf("missing requester in ChannelOpenRequest")
	}
	var addr address.Participant
	err := addr.UnmarshalBinary(requester)
	if err != nil {
		return nil, err
	}
	usr, err := c.UserRegister.GetUserFromParticipant(addr)
	if err == nil {
		return usr, nil
	}
	if !errors.Is(err, ErrUserNotFound) { // TODO: Maybe we should create a new user in this case.
		return nil, err
	}
	usr, err = c.InitializeUser(addr)
	if err != nil {
		return nil, err
	}
	return c.UserRegister.AddUser(addr, usr), nil
}

func (c ChannelService) InitializeUser(participant address.Participant) (*User, error) {
	wAddr, err := c.MakeDefaultWireAddress(participant)
	if err != nil {
		return nil, err
	}
	rs := wallet.NewRemoteSigner(c.wsc, c.ToCKBAddress(participant))
	ckbClient, err := client.NewClient(c.node, rs, c.deployment)
	if err != nil {
		return nil, err
	}
	f := funder.NewDefaultFunder(ckbClient, c.deployment)
	adj := adjudicator.NewAdjudicator(ckbClient)
	watcher, err := local.NewWatcher(adj)
	if err != nil {
		return nil, err
	}
	return NewUser(participant, wAddr, c.bus, f, adj, c.wallet, watcher, c.wsc)
}

func (c ChannelService) GetAllocationFromChannelOpenRequest(request *proto.ChannelOpenRequest) (*channel.Allocation, error) {
	if request.GetAllocation() == nil {
		return nil, fmt.Errorf("missing allocation in ChannelOpenRequest")
	}
	return protobuf.ToAllocation(request.GetAllocation())
}

func (c ChannelService) GetPeerAddressFromChannelOpenRequest(request *proto.ChannelOpenRequest) wire.Address {
	// NOTE: The peer address should probably be a string-encoded CKB Address (see MakeDefaultWireAddress).
	return simple.NewAddress(string(request.GetPeer()))
}

func (c ChannelService) GetChallengeDurationFromChannelOpenRequest(request *proto.ChannelOpenRequest) uint64 {
	return request.ChallengeDuration
}

func (c ChannelService) MakeDefaultWireAddress(participant address.Participant) (wire.Address, error) {
	ckbAddr, err := c.ToCKBAddress(participant).Encode()
	if err != nil {
		return nil, err
	}
	return simple.NewAddress(ckbAddr), nil
}

func (c ChannelService) ToCKBAddress(addr address.Participant) address2.Address {
	return addr.ToCKBAddress(c.network)
}
