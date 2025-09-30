package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	address2 "github.com/nervosnetwork/ckb-sdk-go/v2/address"
	"github.com/nervosnetwork/ckb-sdk-go/v2/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"github.com/perun-network/perun-libp2p-wire/p2p"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/channel/persistence/keyvalue"
	gpwallet "perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/protobuf"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/channel/adjudicator"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/channel/funder"
	"perun.network/perun-ckb-backend/client"
	"perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/wallet/external"
	"polycry.pt/poly-go/sortedkv"
)

const (
	wirePrivateKey = "wire-account-private-key"
)

// ChannelService is the service for handling perun channel operations.
type ChannelService struct {
	user       *User
	wsc        proto.WalletServiceClient
	net        *p2p.Net
	network    types.Network
	node       rpc.Client
	deployment backend.Deployment
	wallet     gpwallet.Wallet

	wireAddr wire.Address
	resolver AddressResolver
	pr       persistence.PersistRestorer

	proto.UnimplementedChannelServiceServer // always embed
}

// NewChannelService creates a new ChannelService.
func NewChannelService(c proto.WalletServiceClient, network types.Network, nodeURL string, deployment backend.Deployment, res AddressResolver, db sortedkv.Database) (*ChannelService, error) {
	node, err := rpc.Dial(nodeURL)
	if err != nil {
		return nil, err
	}

	var wireAcc *p2p.Account
	if b, err := db.Has(wirePrivateKey); err != nil {
		return nil, err
	} else if b {
		wirePrivateKeyBytes, err := db.GetBytes(wirePrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error getting wire account private key: %w", err)
		}

		wireAcc, err = p2p.NewAccountFromPrivateKeyBytes(wirePrivateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("error creating wire account from private key: %w", err)
		}
	} else {
		wireAcc = p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))

		privKeyBytes, err := wireAcc.MarshalPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("error marshalling wire account private key: %w", err)
		}
		err = db.PutBytes(wirePrivateKey, privKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("error storing wire account private key: %w", err)

		}
	}

	wireNet, err := p2p.NewP2PBus(wireAcc)
	if err != nil {
		return nil, fmt.Errorf("error creating wire net: %w", err)
	}

	go wireNet.Bus.Listen(wireNet.Listener)

	if res == nil {
		res = NewRelayServerResolver(wireAcc)
	}

	pr := keyvalue.NewPersistRestorer(db)

	ps, err := pr.ActivePeers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting active peers: %w", err)
	}

	for _, p := range ps {
		// Register peers' LIBP2P address from persistence.
		peerAddr, ok := p.(*p2p.Address)
		if !ok {
			return nil, errors.New("peer address is not a libp2p address")
		}
		wireNet.Dialer.Register(p, peerAddr.String())
	}

	cs := &ChannelService{
		wsc:        c,
		net:        wireNet,
		network:    network,
		node:       node,
		deployment: deployment,
		wallet:     external.NewWallet(wallet.NewExternalClient(c)),
		wireAddr:   wireAcc.Address(),
		resolver:   res,
		pr:         pr,
	}

	return cs, nil
}

// OpenChannel opens a channel for the user.
func (c ChannelService) OpenChannel(ctx context.Context, request *proto.ChannelOpenRequest) (*proto.ChannelOpenResponse, error) {
	log.Println("Received channel open request")
	user, err := c.GetUserFromChannelOpenRequest(request)

	if err != nil {
		return nil, err
	}
	allocation, err := c.GetAllocationFromChannelOpenRequest(request)
	if err != nil {
		return nil, err
	}
	log.Printf("Allocation received: %v", allocation.Balances)
	peer, err := c.GetPeerAddressFromChannelOpenRequest(request)
	if err != nil {
		return nil, err
	}

	// Register peer LIBP2P address
	peerLibp2pAddr, ok := peer.(*p2p.Address)
	if !ok {
		return nil, fmt.Errorf("peer address is not a libp2p address")
	}
	c.net.Dialer.Register(peer, peerLibp2pAddr.String())

	challengeDuration := c.GetChallengeDurationFromChannelOpenRequest(request)
	log.Printf("about to open channel with peer")
	id, err := user.OpenChannel(ctx, peer, allocation, challengeDuration)
	log.Println("Opening request returned")
	if err != nil {
		return &proto.ChannelOpenResponse{Msg: &proto.ChannelOpenResponse_Rejected{Rejected: &proto.Rejected{Reason: err.Error()}}}, err
	}
	return &proto.ChannelOpenResponse{Msg: &proto.ChannelOpenResponse_ChannelId{ChannelId: id[:]}}, nil
}

// UpdateChannel updates the channel for the user.
func (c ChannelService) UpdateChannel(ctx context.Context, request *proto.ChannelUpdateRequest) (*proto.ChannelUpdateResponse, error) {
	log.Println("Channel Service received update request")
	cid, user, err := c.GetChannelInfoFromRequest(request.State.GetId())
	if err != nil {
		return nil, err
	}
	state, err := AsChannelState(request.GetState())
	if err != nil {
		return nil, err
	}
	newState, err := user.UpdateChannel(ctx, cid, state)
	if err != nil {
		if newState != nil {
			panic("newState should be nil on error")
		}
		rejected := proto.Rejected{Reason: err.Error()}
		return &proto.ChannelUpdateResponse{Msg: &proto.ChannelUpdateResponse_Rejected{Rejected: &rejected}}, err
	}

	newStateProto, err := protobuf.FromState(newState)
	if err != nil {
		return nil, err
	}

	return &proto.ChannelUpdateResponse{Msg: &proto.ChannelUpdateResponse_Update{Update: &proto.SuccessfulUpdate{
		State:     newStateProto,
		ChannelId: cid[:],
	}}}, nil
}

// CloseChannel closes the channel for the user.
func (c ChannelService) CloseChannel(ctx context.Context, request *proto.ChannelCloseRequest) (*proto.ChannelCloseResponse, error) {
	cid, user, err := c.GetChannelInfoFromRequest(request.GetChannelId())
	if err != nil {
		return nil, err
	}
	err = user.CloseChannel(ctx, cid)
	if err != nil {
		return &proto.ChannelCloseResponse{Msg: &proto.ChannelCloseResponse_Rejected{Rejected: &proto.Rejected{Reason: err.Error()}}}, err
	}
	return &proto.ChannelCloseResponse{Msg: &proto.ChannelCloseResponse_Close{Close: &proto.SuccessfulClose{ChannelId: cid[:]}}}, nil
}

// GetChannels returns the channels for the user.
func (c ChannelService) GetChannels(ctx context.Context, request *proto.GetChannelsRequest) (*proto.GetChannelsResponse, error) {
	u, err := c.getUserFromGetChannelsRequest(request)
	if err != nil {
		return nil, err
	}
	states, actorIndexes := u.GetChannels()
	if len(states) == 0 {
		return &proto.GetChannelsResponse{Msg: &proto.GetChannelsResponse_Rejected{Rejected: &proto.Rejected{Reason: "no channels exists for user"}}}, nil
	}
	pStates := make([]*protobuf.State, len(states))
	pActorIndexes := make([]uint32, len(actorIndexes))
	for i, state := range states {
		pState, err := protobuf.FromState(&state)
		if err != nil {
			return nil, fmt.Errorf("error converting state to protobuf: %w", err)
		}
		pStates[i] = pState
		pActorIndexes[i] = uint32(actorIndexes[i])
	}
	channelStates := &proto.ChannelStates{
		States:    pStates,
		ActorIdxs: pActorIndexes,
	}

	return &proto.GetChannelsResponse{Msg: &proto.GetChannelsResponse_ChannelStates{ChannelStates: channelStates}}, nil
}

func (c ChannelService) getUserFromGetChannelsRequest(request *proto.GetChannelsRequest) (*User, error) {
	r := request.GetRequester()
	if r == nil {
		return nil, fmt.Errorf("missing requester in GetChannelsRequest")
	}
	var addr address.Participant
	err := addr.UnmarshalBinary(r)
	if err != nil {
		return nil, err
	}

	if c.user != nil {
		if c.user.Participant.Equal(&addr) {
			return c.user, nil
		}
	}

	return nil, fmt.Errorf("user %s not found", addr)
}

// AsChannelID converts a byte slice to a channel ID.
func AsChannelID(in []byte) (channel.ID, error) {
	id := channel.ID{}
	n := copy(id[:], in)
	if n != len(id) {
		return channel.ID{}, fmt.Errorf("channel id too short: expected %d bytes, got %d", len(id), n)
	}
	return id, nil
}

// AsChannelState converts a protobuf state to a channel state.
func AsChannelState(ps *protobuf.State) (*channel.State, error) {
	log.Println("Converting protobuf state to channel state")
	return protobuf.ToState(ps)
}

// GetChannelInfoFromRequest returns the channel ID and user from the request.
func (c ChannelService) GetChannelInfoFromRequest(reqChannelId []byte) (channel.ID, *User, error) {
	cid, err := AsChannelID(reqChannelId)
	if err != nil {
		return channel.ID{}, nil, err
	}
	if c.user == nil {
		return channel.ID{}, nil, fmt.Errorf("user not found")
	}
	return cid, c.user, err
}

// GetUserFromChannelOpenRequest returns the user from the channel open request.
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

	log.Printf("Participant to fetch: %s", addr)

	if c.user == nil {
		log.Printf("User not found, initializing user %s", addr)
		_, err := c.InitializeUser(addr, c.wsc, c.wallet)
		if err != nil {
			return nil, err

		}
		return c.user, nil
	}

	if !c.user.Participant.Equal(&addr) {
		return nil, fmt.Errorf("user %s does not match requester %s", c.user.Participant, addr)
	}

	return c.user, nil
}

// InitializeUser initializes a user with the given participant.
func (c *ChannelService) InitializeUser(participant address.Participant, wsc proto.WalletServiceClient, w gpwallet.Wallet) (*User, error) {
	log.Printf("Initializing user %s", participant)

	wAddr, err := c.SetWireAddress(participant)
	if err != nil {
		return nil, err
	}
	rs := wallet.NewRemoteSigner(wsc, c.ToCKBAddress(participant))
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
	usr, err := NewUser(participant, wAddr, c.net.Bus, f, adj, w, watcher, wsc, c.pr)
	if err != nil {
		return nil, err
	}
	c.user = usr
	return usr, nil
}

func (c ChannelService) GetAllocationFromChannelOpenRequest(request *proto.ChannelOpenRequest) (*channel.Allocation, error) {
	if request.GetAllocation() == nil {
		return nil, fmt.Errorf("missing allocation in ChannelOpenRequest")
	}
	return toCKBAllocation(request.GetAllocation())
}

func toCKBAllocation(protoAlloc *protobuf.Allocation) (*channel.Allocation, error) {
	alloc := &channel.Allocation{}
	alloc.Assets = make([]channel.Asset, len(protoAlloc.Assets))
	for i := range protoAlloc.Assets {
		// NOTE: We will assume the first asset will always be CKBytes.
		if i == 0 {
			alloc.Assets[i] = &asset.Asset{
				IsCKBytes: true,
				SUDT:      nil,
			}
		} else {
			alloc.Assets[i] = channel.NewAsset()
		}
		err := alloc.Assets[i].UnmarshalBinary(protoAlloc.Assets[i])
		if err != nil {
			return nil, fmt.Errorf("%d'th asset: %w", i, err)
		}
	}
	alloc.Locked = make([]channel.SubAlloc, len(protoAlloc.Locked))
	for i := range protoAlloc.Locked {
		locked, err := protobuf.ToSubAlloc(protoAlloc.Locked[i])
		if err != nil {
			return nil, fmt.Errorf("%d'th sub alloc: %w", i, err)
		}
		alloc.Locked[i] = locked
	}
	alloc.Balances = protobuf.ToBalances(protoAlloc.Balances)

	return alloc, nil
}

// GetPeerAddressFromChannelOpenRequest returns the peer address from the channel open request.
func (c ChannelService) GetPeerAddressFromChannelOpenRequest(request *proto.ChannelOpenRequest) (wire.Address, error) {
	// NOTE: The peer address should probably be a string-encoded CKB Address (see MakeDefaultWireAddress).
	peer := request.GetPeer()
	if peer == nil {
		return nil, fmt.Errorf("missing requester in ChannelOpenRequest")
	}

	var addr address.Participant
	err := addr.UnmarshalBinary(peer)
	if err != nil {
		return nil, err
	}

	return c.resolver.GetWireAddress(&addr)
}

func (c ChannelService) GetChallengeDurationFromChannelOpenRequest(request *proto.ChannelOpenRequest) uint64 {
	return request.ChallengeDuration
}

// SetWireAddress sets the wire address for the given participant.
func (c ChannelService) SetWireAddress(participant address.Participant) (wire.Address, error) {
	return c.wireAddr, c.resolver.SetWire(&participant, c.wireAddr)
}

// ToCKBAddress converts a participant address to a CKB address.
func (c ChannelService) ToCKBAddress(addr address.Participant) address2.Address {
	return addr.ToCKBAddress(c.network)
}

// ClosePerunClient closes the Perun client for the user.
func (c ChannelService) ClosePerunClient(ctx context.Context, req *proto.ClosePerunClientRequest) (*proto.ClosePerunClientResponse, error) {
	// Delete the wire
	err := c.resolver.DeleteWire(&c.user.Participant)
	if err != nil {
		log.Fatalf("Error deleting wire: %v", err)
		return nil, err
	}

	err = c.user.PerunClient.Close()
	if err != nil {
		log.Fatalf("Error closing perun client: %v", err)
		return nil, err
	}
	c.user.Channels = nil
	return &proto.ClosePerunClientResponse{}, nil
}

// NewPerunClient creates a new Perun client for the user.
func (c ChannelService) NewPerunClient(ctx context.Context, request *proto.NewPerunClientRequest) (*proto.NewPerunClientResponse, error) {
	addr := c.user.Participant
	if c.user == nil {
		log.Fatalf("User not found")
		return &proto.NewPerunClientResponse{Accepted: false}, errors.New("user not found")
	}
	wAddr := c.wireAddr
	rs := wallet.NewRemoteSigner(c.wsc, c.ToCKBAddress(addr))
	ckbClient, err := client.NewClient(c.node, rs, c.deployment)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
		return &proto.NewPerunClientResponse{Accepted: false}, err
	}
	f := funder.NewDefaultFunder(ckbClient, c.deployment)
	adj := adjudicator.NewAdjudicator(ckbClient)
	watcher, err := local.NewWatcher(adj)
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
		return &proto.NewPerunClientResponse{Accepted: false}, err
	}
	c.user.NewPerunClient(wAddr, c.net.Bus, f, adj, c.wallet, watcher, c.wsc, c.pr)
	return &proto.NewPerunClientResponse{Accepted: true}, nil
}

// RestoreChannels restores the channels for the user.
func (c ChannelService) RestoreChannels(ctx context.Context, _ *proto.RestoreChannelsRequest) (*proto.RestoreChannelsResponse, error) {
	err := c.user.RestoreChannels(ctx)
	if err != nil {
		return &proto.RestoreChannelsResponse{Accepted: false}, err
	}
	return &proto.RestoreChannelsResponse{Accepted: true}, nil
}

func (c ChannelService) Close() error {
	return c.net.Bus.Close()
}
