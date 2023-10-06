package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"perun.network/channel-service/rpc"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/wallet/address"
	"sync"
)

var ErrChannelNotFound = errors.New("channel not found")

// User handles all channel related operations for a single user (wire / wallet address pair).
type User struct {
	usrMutex sync.Mutex

	Channels    map[channel.ID]*client.Channel
	Participant address.Participant
	PerunClient *client.Client
	WireAddress wire.Address
	wsc         rpc.WalletServiceClient
}

func (u *User) HandleUpdate(oldState *channel.State, update client.ChannelUpdate, responder *client.UpdateResponder) {
	pbNewState, err := protobuf.FromState(update.State.Clone())
	if err != nil {
		_ = responder.Reject(context.TODO(), "unable to encode state")
		return
	}

	resp, err := u.wsc.UpdateNotification(context.TODO(), &rpc.UpdateNotificationRequest{
		State:     pbNewState,
		ChannelId: pbNewState.Id,
	})
	if err != nil {
		_ = responder.Reject(context.TODO(), "unable to send update notification to wallet")
		return
	}
	if resp.GetAccepted() {
		_ = responder.Accept(context.TODO())
	} else {
		_ = responder.Reject(context.TODO(), "wallet rejected update")
	}

}

func (u *User) HandleProposal(proposal client.ChannelProposal, responder *client.ProposalResponder) {
	proposal.Base()
	// FIXME: Make protobuf type for client.ChannelProposal and use it in OpenChannel.
	panic("fixme")
	// FIXME: OpenChannel needs OpenChannelResponse as return type!
	// wcs.OpenChannel(nil, nil)
}

func (u *User) HandleAdjudicatorEvent(event channel.AdjudicatorEvent) {
	// TODO: Do we need to do anything here?
	log.Printf("Adjudicator event: type = %T", event)
}

func NewUser(participant address.Participant, wAddr wire.Address, bus wire.Bus, funder channel.Funder, adjudicator channel.Adjudicator, wallet wallet.Wallet, watcher watcher.Watcher, wsc rpc.WalletServiceClient) (*User, error) {
	c, err := client.New(wAddr, bus, funder, adjudicator, wallet, watcher)
	if err != nil {
		return nil, err
	}
	u := &User{
		Participant: participant,
		PerunClient: c,
		WireAddress: wAddr,
		wsc:         wsc,
	}
	go c.Handle(u, u)
	return u, nil
}

func (u *User) OpenChannel(ctxt context.Context, peer wire.Address, allocation *channel.Allocation, challengeDuration uint64) (channel.ID, error) {
	proposal, err := client.NewLedgerChannelProposal(
		challengeDuration,
		&u.Participant,
		allocation,
		[]wire.Address{u.WireAddress})
	if err != nil {
		return channel.ID{}, err
	}
	ch, err := u.PerunClient.ProposeChannel(ctxt, proposal)
	if err != nil {
		return channel.ID{}, err
	}
	u.startWatching(ch)
	u.usrMutex.Lock()
	defer u.usrMutex.Unlock()
	u.Channels[ch.ID()] = ch
	return ch.ID(), nil
}

func (u *User) UpdateChannel(ctxt context.Context, id channel.ID, alloc channel.Allocation) error {
	u.usrMutex.Lock()
	defer u.usrMutex.Unlock()
	ch, ok := u.Channels[id]
	if !ok {
		return ErrChannelNotFound
	}
	err := ch.Update(ctxt, UpdateToAllocation(alloc))
	return err
}

func UpdateToAllocation(alloc channel.Allocation) func(state *channel.State) {
	return func(state *channel.State) {
		// TODO: Properly update allocation with checks etc.
		state.Allocation = alloc
	}
}

func (u *User) CloseChannel(ctxt context.Context, id channel.ID) error {
	u.usrMutex.Lock()
	defer u.usrMutex.Unlock()
	ch, ok := u.Channels[id]
	if !ok {
		return ErrChannelNotFound
	}
	// Finalize the channel to enable fast settlement.
	if !ch.State().IsFinal {
		err := ch.Update(ctxt, func(state *channel.State) {
			state.IsFinal = true
		})
		if err != nil {
			panic(err)
		}
	}

	// Settle concludes the channel and withdraws the funds.
	err := ch.Settle(ctxt)
	if err != nil {
		panic(err)
	}

	// Close frees up channel resources.
	_ = ch.Close()
	delete(u.Channels, id)
	return nil
}

// startWatching starts the dispute watcher for the specified channel.
func (u *User) startWatching(ch *client.Channel) {
	go func() {
		err := ch.Watch(u)
		if err != nil {
			fmt.Printf("Watcher returned with error: %v", err)
		}
	}()
}
