package service

import (
	"context"
	"fmt"
	"perun.network/channel-service/rpc"
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

	rpc.UnimplementedChannelServiceServer // always embed
}

func NewChannelService(c rpc.WalletServiceClient) *ChannelService {
	return &ChannelService{
		UserRegister: NewMutexUserRegister(),
		Wallet:       external.NewWallet(wallet.NewExternalClient(c)),
	}
}

func (c ChannelService) OpenChannel(ctx context.Context, request *rpc.ChannelOpenRequest) (*rpc.ChannelOpenResponse, error) {

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
		return &rpc.ChannelOpenResponse{Msg: &rpc.ChannelOpenResponse_Rejected{Rejected: &rpc.Rejected{Reason: err.Error()}}}, err
	}
	return &rpc.ChannelOpenResponse{Msg: &rpc.ChannelOpenResponse_ChannelId{ChannelId: id[:]}}, nil

}

func (c ChannelService) UpdateChannel(ctx context.Context, request *rpc.ChannelUpdateRequest) (*rpc.ChannelUpdateResponse, error) {
	// FIXME: rpc.UpdateChannelRequest should probably not contain a state. If it does, we need to verify that it matches the state in the channel.
	cid, user, err := c.GetChannelInfoFromRequest(request.GetChannelId())
	if err != nil {
		return nil, err
	}
	state, err := AsChannelState(request.GetState())
	if err != nil {
		return nil, err
	}
	err = user.UpdateChannel(ctx, cid, state.Allocation)
	if err != nil {
		rejected := rpc.Rejected{Reason: err.Error()}
		return &rpc.ChannelUpdateResponse{Msg: &rpc.ChannelUpdateResponse_Rejected{Rejected: &rejected}}, err
	}

	return &rpc.ChannelUpdateResponse{Msg: &rpc.ChannelUpdateResponse_Update{Update: &rpc.SuccessfulUpdate{
		// TODO: Use actual resulting state instead of the request state.
		State: request.State,
		// TODO: Abstract channel id encoding.
		ChannelId: cid[:],
	}}}, nil
}

func (c ChannelService) CloseChannel(ctx context.Context, request *rpc.ChannelCloseRequest) (*rpc.ChannelCloseResponse, error) {
	cid, user, err := c.GetChannelInfoFromRequest(request.GetChannelId())
	if err != nil {
		return nil, err
	}
	err = user.CloseChannel(ctx, cid)
	if err != nil {
		// TODO: Do we want to return the error here?
		return &rpc.ChannelCloseResponse{Msg: &rpc.ChannelCloseResponse_Rejected{Rejected: &rpc.Rejected{Reason: err.Error()}}}, err
	}
	return &rpc.ChannelCloseResponse{Msg: &rpc.ChannelCloseResponse_Close{Close: &rpc.SuccessfulClose{ChannelId: cid[:]}}}, nil
}

func (c ChannelService) ForceCloseChannel(ctx context.Context, request *rpc.ChannelForceCloseRequest) (*rpc.ChannelForceCloseResponse, error) {
	cid, user, err := c.GetChannelInfoFromRequest(request.GetChannelId())
	if err != nil {
		return nil, err
	}
	// TODO: Verify assumption that this is the same as close.
	err = user.CloseChannel(ctx, cid)
	if err != nil {
		return &rpc.ChannelForceCloseResponse{Msg: &rpc.ChannelForceCloseResponse_Rejected{Rejected: &rpc.Rejected{Reason: err.Error()}}}, err
	}
	return &rpc.ChannelForceCloseResponse{Msg: &rpc.ChannelForceCloseResponse_Close{Close: &rpc.SuccessfulClose{ChannelId: cid[:]}}}, nil
}

func (c ChannelService) ChallengeChannel(ctx context.Context, request *rpc.ChallengeChannelRequest) (*rpc.ChallengeChannelResponse, error) {
	// FIXME: This endpoint does not make sense. It should be removed.
	// There is no concept of challenging with an arbitrary state, but only with the latest state.
	// In the context of Payment Channels, this is the same as settle.

	panic("fixme")
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

func (c ChannelService) GetUserFromChannelOpenRequest(request *rpc.ChannelOpenRequest) (*User, error) {
	// FIXME: The OpenChannelRequest should contain info about the acting participant.
	panic("fixme")
}

func (c ChannelService) GetAllocationFromChannelOpenRequest(request *rpc.ChannelOpenRequest) (*channel.Allocation, error) {
	// FIXME: ChannelOpenRequest should specify an allocation / initial funds.
	panic("fixme")
}

func (c ChannelService) GetPeerAddressFromChannelOpenRequest(request *rpc.ChannelOpenRequest) wire.Address {
	// FIXME: Do this properly and change the peer type in the request field.
	return simple.NewAddress(string(request.GetPeer()))
}

func (c ChannelService) GetChallengeDurationFromChannelOpenRequest(request *rpc.ChannelOpenRequest) uint64 {
	// TODO: This should be specified in the request or hardcoded with default value.
	return 10
}
