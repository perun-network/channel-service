package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/wallet/address"
)

var ErrChannelNotFound = errors.New("channel not found")

// User handles all channel related operations for a single user (wire / wallet address pair).
type User struct {
	usrMutex sync.Mutex

	Channels     map[channel.ID]*client.Channel
	Participant  address.Participant
	PerunClient  *client.Client
	WireAddress  wire.Address
	wsc          proto.WalletServiceClient
	userRegister UserRegister
}

func (u *User) HandleUpdate(oldState *channel.State, update client.ChannelUpdate, responder *client.UpdateResponder) {
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

func (u *User) HandleProposal(proposal client.ChannelProposal, responder *client.ProposalResponder) {
	addr, err := u.Participant.ToCKBAddress(types.NetworkTest).Encode()
	if err != nil {
		panic(fmt.Sprintf("encoding participant addr: %v", err))
	}
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
	err = u.userRegister.AssignChannelID(ch.ID(), u)
	if err != nil {
		panic(err)
	}
	u.Channels[ch.ID()] = ch
}

func (u *User) HandleAdjudicatorEvent(event channel.AdjudicatorEvent) {
	// TODO: Do we need to do anything here?
	// TODO: Inform wallet service server about event.
	log.Printf("Adjudicator event: type = %T", event)
}

func NewUser(participant address.Participant, wAddr wire.Address, bus wire.Bus, funder channel.Funder, adjudicator channel.Adjudicator, wallet wallet.Wallet, watcher watcher.Watcher, wsc proto.WalletServiceClient, reg UserRegister) (*User, error) {
	c, err := client.New(wAddr, bus, funder, adjudicator, wallet, watcher)
	if err != nil {
		return nil, err
	}
	u := &User{
		Participant:  participant,
		PerunClient:  c,
		WireAddress:  wAddr,
		wsc:          wsc,
		userRegister: reg,
		Channels:     make(map[channel.ID]*client.Channel),
	}
	go c.Handle(u, u)
	return u, nil
}

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
	u.startWatching(ch)
	u.usrMutex.Lock()
	defer u.usrMutex.Unlock()
	u.Channels[ch.ID()] = ch
	return ch.ID(), nil
}

func (u *User) UpdateChannel(ctxt context.Context, id channel.ID, newState *channel.State) error {
	u.usrMutex.Lock()
	defer u.usrMutex.Unlock()
	ch, ok := u.Channels[id]
	if !ok {
		return ErrChannelNotFound
	}
	if err := VerifyStateTransition(ch.State().Clone(), newState.Clone()); err != nil {
		return err
	}
	err := ch.Update(ctxt, UpdateToState(newState))
	return err
}

func VerifyStateTransition(old, new *channel.State) error {
	// TODO: implement
	return nil
}

func UpdateToAllocation(alloc channel.Allocation) func(state *channel.State) {
	return func(state *channel.State) {
		// TODO: Properly update allocation with checks etc.
		state.Allocation = alloc
	}
}

func UpdateToState(ns *channel.State) func(state *channel.State) {
	return func(state *channel.State) {
		*state = *ns
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

func (u *User) GetChannels() []channel.State {
	// TODO: Consider concurrency issues.
	var states []channel.State
	for _, ch := range u.Channels {
		states = append(states, *ch.State().Clone())
	}
	return states
}
