package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/wallet/address"
)

// ErrChannelNotFound is returned when a channel with the specified ID is not found.
var ErrChannelNotFound = errors.New("channel not found")

// User handles all channel related operations for a single user (wire / wallet address pair).
type User struct {
	Channels    map[channel.ID]*client.Channel
	Participant address.Participant
	PerunClient *client.Client
	WireAddress wire.Address
	wsc         proto.WalletServiceClient
}

// HandleUpdate handles a channel update.
func (u *User) HandleUpdate(_ *channel.State, update client.ChannelUpdate, responder *client.UpdateResponder) {
	pbNewState, err := protobuf.FromState(update.State.Clone())
	if err != nil {
		_ = responder.Reject(context.TODO(), "unable to encode state")
		return
	}

	resp, err := u.wsc.UpdateNotification(context.TODO(), &proto.UpdateNotificationRequest{
		State: pbNewState,
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

// HandleProposal handles a channel proposal.
func (u *User) HandleProposal(proposal client.ChannelProposal, responder *client.ProposalResponder) { // ASSUMPTION: the responder parameter contains the same perun client which this user has
	addr, err := u.Participant.ToCKBAddress(types.NetworkTest).Encode()
	if err != nil {
		panic(fmt.Sprintf("encoding participant addr: %v", err))
	}
	log.Printf("Handling channel proposal as user: %s", u.Participant)
	log.Printf("Handling channel proposal as user: %s", addr)
	lcp, ok := proposal.(*client.LedgerChannelProposalMsg)
	if !ok {
		_ = responder.Reject(context.TODO(), "only ledger channel proposals are supported")
		return
	}
	pLcp, err := protobuf.FromLedgerChannelProposalMsg(lcp)
	if err != nil {
		_ = responder.Reject(context.TODO(), fmt.Sprintf("unable to encode proposal: %v", err))
		return
	}
	log.Println("Requesting nonce share from wallet")
	resp, err := u.wsc.OpenChannel(context.TODO(), &proto.OpenChannelRequest{Proposal: pLcp.LedgerChannelProposalMsg})
	if err != nil {
		_ = responder.Reject(context.TODO(), fmt.Sprintf("unable to open channel: %v", err))
		return
	}
	log.Println("Received nonce share from wallet")
	ns := resp.GetNonceShare()
	if ns == nil {
		if resp.GetRejected() != nil {
			_ = responder.Reject(context.TODO(), resp.GetRejected().GetReason())
			return
		} else {
			_ = responder.Reject(context.TODO(), "wallet rejected channel proposal")
			return
		}
	}
	nonceShare := client.NonceShare{}
	copy(nonceShare[:], ns)
	cpa := client.LedgerChannelProposalAccMsg{
		BaseChannelProposalAcc: client.BaseChannelProposalAcc{
			ProposalID: lcp.ProposalID,
			NonceShare: nonceShare,
		},
		Participant: &u.Participant,
	}
	ch, err := responder.Accept(context.TODO(), &cpa)
	if err != nil {
		panic(err)
	}
	u.Channels[ch.ID()] = ch
	u.startWatching(ch)
	ch.OnUpdate(u.NotifyAllState)
	u.NotifyAllState(nil, ch.State())

}

// HandleAdjudicatorEvent handles an adjudicator event.
func (u *User) HandleAdjudicatorEvent(event channel.AdjudicatorEvent) {
	// TODO: Do we need to do anything here?
	// TODO: Inform wallet service server about event.
	log.Printf("Adjudicator event: type = %T\n", event)
}

// NewUser creates a new user with the specified participant, wire address, bus, funder, adjudicator, wallet, watcher, wallet service client and persistence.
func NewUser(participant address.Participant, wAddr wire.Address, bus wire.Bus, funder channel.Funder, adjudicator channel.Adjudicator, wallet wallet.Wallet, watcher watcher.Watcher, wsc proto.WalletServiceClient, pr persistence.PersistRestorer) (*User, error) {
	c, err := client.New(wAddr, bus, funder, adjudicator, wallet, watcher)
	c.EnablePersistence(pr) //automatically saves channels to persistence
	if err != nil {
		return nil, err
	}
	u := &User{
		Participant: participant,
		PerunClient: c,
		WireAddress: wAddr,
		wsc:         wsc,
		Channels:    make(map[channel.ID]*client.Channel),
	}
	go c.Handle(u, u)
	return u, nil
}

// NewPerunClient creates a new Perun client for the user.
func (u *User) NewPerunClient(wAddr wire.Address, bus wire.Bus, funder channel.Funder, adjudicator channel.Adjudicator, wallet wallet.Wallet, watcher watcher.Watcher, wsc proto.WalletServiceClient, pr persistence.PersistRestorer) {
	perunClient, err := client.New(wAddr, bus, funder, adjudicator, wallet, watcher)
	if err != nil {
		log.Printf("Erro creating new client for user: %v", err)
		panic(err)
	}
	perunClient.EnablePersistence(pr)
	go perunClient.Handle(u, u)
	u.PerunClient = perunClient
}

// RestoreChannels restores all channels for the user.
func (u *User) RestoreChannels(ctx context.Context) error {
	// Restore all channels for this user.
	channels := make(map[channel.ID]*client.Channel)

	u.PerunClient.OnNewChannel(func(ch *client.Channel) {
		u.startWatching(ch)
		ch.OnUpdate(u.NotifyAllState)
		u.NotifyAllState(nil, ch.State())
		channels[ch.ID()] = ch
	})

	err := u.PerunClient.Restore(ctx)
	if err != nil {
		log.Fatalf("Error restoring channels in user.go: %v", err)
		return err
	}
	u.Channels = channels
	return nil
}

// OpenChannel opens a new channel with the specified peer.
func (u *User) OpenChannel(ctxt context.Context, peer wire.Address, allocation *channel.Allocation, challengeDuration uint64) (channel.ID, error) {
	proposal, err := client.NewLedgerChannelProposal(
		challengeDuration,
		&u.Participant,
		allocation,
		[]wire.Address{u.WireAddress, peer})
	if err != nil {
		return channel.ID{}, fmt.Errorf("creating LedgerChannelProposal: %w", err)
	}
	log.Println("Proposing channel on PerunClient")
	ch, err := u.PerunClient.ProposeChannel(ctxt, proposal)
	if err != nil {
		return channel.ID{}, fmt.Errorf("proposing channel: %w", err)
	}
	ch.OnUpdate(u.NotifyAllState)
	u.startWatching(ch)
	u.Channels[ch.ID()] = ch
	u.NotifyAllState(nil, ch.State())
	return ch.ID(), nil
}

// UpdateChannel updates the channel with the specified ID to the new state.
func (u *User) UpdateChannel(ctxt context.Context, id channel.ID, newState *channel.State) (*channel.State, error) {
	ch, ok := u.Channels[id]
	if !ok {
		return nil, ErrChannelNotFound
	}
	if err := VerifyStateTransition(ch.State().Clone(), newState.Clone()); err != nil {
		return nil, err
	}

	log.Println("Updating channel on PerunClient")
	err := ch.Update(ctxt, UpdateToState(newState))

	return u.Channels[id].State(), err
}

// VerifyStateTransition verifies that the transition from the old state to the new state is valid.
func VerifyStateTransition(old, new *channel.State) error {
	// TODO: implement
	return nil
}

// UpdateToAllocation returns a function that updates the state of a channel to the specified allocation.
func UpdateToAllocation(alloc channel.Allocation) func(state *channel.State) {
	return func(state *channel.State) {
		// TODO: Properly update allocation with checks etc.
		state.Allocation = alloc
	}
}

// UpdateToState returns a function that updates the state of a channel to the specified state.
func UpdateToState(ns *channel.State) func(state *channel.State) {
	return func(state *channel.State) {
		*state = *ns
	}
}

// CloseChannel closes the channel with the specified ID.
func (u *User) CloseChannel(ctxt context.Context, id channel.ID) error {
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
	err := ch.Settle(ctxt, false)
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

// GetChannels returns the current state of all channels.
func (u *User) GetChannels() ([]channel.State, []channel.Index) {
	var states []channel.State
	var actorIndexes []channel.Index
	for _, ch := range u.Channels {
		states = append(states, *ch.State().Clone())
		actorIndexes = append(actorIndexes, ch.Idx())
	}
	return states, actorIndexes
}

// NotifyAllState notifies the wallet service about the new state of the channel.
func (u *User) NotifyAllState(_, to *channel.State) {
	pbNewState, err := protobuf.FromState(to.Clone())
	if err != nil {
		panic(fmt.Sprintf("unable to encode state: %v", err))
	}

	resp, err := u.wsc.UpdateNotification(context.TODO(), &proto.UpdateNotificationRequest{
		State: pbNewState,
	})
	if err != nil {
		panic(fmt.Sprintf("unable to send update notification to wallet: %v", err))
	}
	if !resp.GetAccepted() {
		panic("wallet rejected update")
	}
}
