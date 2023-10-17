package service

import (
	"context"
	"fmt"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/wallet"
	"perun.network/go-perun/channel"
	gpwallet "perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/wallet/external"
)

type ChannelService struct {
	UserRegister UserRegister
	Wallet       gpwallet.Wallet

	proto.UnimplementedChannelServiceServer // always embed
}

func NewChannelService(c proto.WalletServiceClient) *ChannelService {
	return &ChannelService{
		UserRegister: NewMutexUserRegister(),
		Wallet:       external.NewWallet(wallet.NewExternalClient(c)),
	}
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
		State: request.State,
		// TODO: Abstract channel id encoding.
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
	// FIXME: The OpenChannelRequest should contain info about the acting participant.
	panic("fixme")
}

func (c ChannelService) GetAllocationFromChannelOpenRequest(request *proto.ChannelOpenRequest) (*channel.Allocation, error) {
	// FIXME: ChannelOpenRequest should specify an allocation / initial funds.
	panic("fixme")
}

func (c ChannelService) GetPeerAddressFromChannelOpenRequest(request *proto.ChannelOpenRequest) wire.Address {
	// FIXME: Do this properly and change the peer type in the request field.
	return simple.NewAddress(string(request.GetPeer()))
}

func (c ChannelService) GetChallengeDurationFromChannelOpenRequest(request *proto.ChannelOpenRequest) uint64 {
	// TODO: This should be specified in the request or hardcoded with default value.
	return 10
}
