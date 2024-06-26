package test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	defaultnet "net"
	"os"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"github.com/perun-network/perun-libp2p-wire/p2p"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"perun.network/channel-service/deployment"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/channel-service/service/test"
	chwallet "perun.network/channel-service/wallet"
	"polycry.pt/poly-go/sortedkv/memorydb"

	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/go-perun/wire"

	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/channel/asset"
	ckbwallet "perun.network/perun-ckb-backend/wallet"
	ckbaddr "perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/wallet/external"
)

const (
	rpcNodeURL = "http://localhost:8114"
	Network    = types.NetworkTest
	devNetDir  = "test/devnet"
	bufSize    = 1024 * 1024
)

//var lis *bufconn.Listener

type Setup struct {
	t                          *testing.T
	Deployment                 backend.Deployment
	SUDTInfo                   deployment.SUDTInfo
	WalletAccs                 []*ckbwallet.Account
	AccNets                    []*p2p.Net
	AccPersistRestorers        []persistence.PersistRestorer
	Asset                      asset.Asset
	AccKeys                    []*secp256k1.PrivateKey
	WireAccs                   []*p2p.Account
	Participants               []ckbaddr.Participant
	WalletServiceClients       []proto.WalletServiceClient
	WscCleanupFuncs            []func()
	WalletServices             []*test.MyWalletService
	ChannelServiceClients      []proto.ChannelServiceClient
	ChannelServiceCleanupFuncs []func()
}

func NewTestSetup(t *testing.T) *Setup {
	setup := &Setup{}
	setup.t = t
	sudtOwnerLockArg, err := parseSUDTOwnerLockArg(devNetDir + "/accounts/sudt-owner-lock-hash.txt")
	require.NoError(t, err, "error getting SUDT owner lock arg")

	d, sudtInfo, err := deployment.GetDeployment(devNetDir+"/contracts/migrations/dev/", devNetDir+"/system_scripts", sudtOwnerLockArg)
	require.NoError(t, err, "error getting deployment")
	setup.Deployment = d
	setup.SUDTInfo = sudtInfo

	alicePrivateKey, err := deployment.GetKey(devNetDir + "/accounts/alice.pk")
	require.NoError(t, err, "error getting alice's private key")

	bobPrivateKey, err := deployment.GetKey(devNetDir + "/accounts/bob.pk")
	require.NoError(t, err, "error getting bob's private key")

	pubKeys := []*secp256k1.PublicKey{alicePrivateKey.PubKey(), bobPrivateKey.PubKey()}
	parts, err := MakeParticipants(pubKeys)
	setup.Participants = parts
	require.NoError(t, err, "error making participants")

	aliceAccount := ckbwallet.NewAccountFromPrivateKey(alicePrivateKey)
	bobAccount := ckbwallet.NewAccountFromPrivateKey(bobPrivateKey)

	setup.WalletAccs = []*ckbwallet.Account{aliceAccount, bobAccount}
	setup.AccKeys = []*secp256k1.PrivateKey{alicePrivateKey, bobPrivateKey}

	aliceWSC, aliceWSCCleanup := setup.setupWalletService(t, "alice", context.Background(), aliceAccount, alicePrivateKey, Network)
	bobWSC, bobWSCCleanup := setup.setupWalletService(t, "bob", context.Background(), bobAccount, bobPrivateKey, Network)

	setup.WscCleanupFuncs = []func(){aliceWSCCleanup, bobWSCCleanup}
	setup.WalletServiceClients = []proto.WalletServiceClient{aliceWSC, bobWSC}

	// AddressRessolver
	ar := service.NewMutexLocalAddressResolver()

	aliceWireAcc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	aliceNet, err := p2p.NewP2PBus(aliceWireAcc)
	require.NoError(t, err, "error creating p2p net")
	go aliceNet.Bus.Listen(aliceNet.Listener)

	bobWireAcc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	bobNet, err := p2p.NewP2PBus(bobWireAcc)
	require.NoError(t, err, "error creating p2p net")
	go bobNet.Bus.Listen(bobNet.Listener)

	setup.WireAccs = []*p2p.Account{aliceWireAcc, bobWireAcc}
	setup.AccNets = []*p2p.Net{aliceNet, bobNet}

	//setup channel-service
	fmt.Printf("aliceWireAcc: %v\n", aliceWireAcc)
	fmt.Printf("bobWireAcc: %v\n", bobWireAcc)
	aliceWireAccAddrString, ok := aliceWireAcc.Address().(*p2p.Address)
	if !ok {
		log.Printf("error casting to p2p.Address")
	}
	log.Printf("aliceWireAccAddrString: %v", aliceWireAccAddrString)
	bobWireAccAddrString, ok := bobWireAcc.Address().(*p2p.Address)
	if !ok {
		log.Printf("error casting to p2p.Address")
	}
	log.Printf("bobWireAccAddrString: %v", bobWireAccAddrString)

	prAlice := keyvalue.NewPersistRestorer(memorydb.NewDatabase())
	prBob := keyvalue.NewPersistRestorer(memorydb.NewDatabase())
	setup.AccPersistRestorers = []persistence.PersistRestorer{prAlice, prBob}

	aliceCSClient, aliceCS, aliceCSCleanup := setupChannelService(t, "alice", aliceWSC, aliceNet, Network, rpcNodeURL, d, aliceWireAcc.Address(), ar, prAlice)
	bobCSClient, bobCS, bobCSCleanup := setupChannelService(t, "bob", bobWSC, bobNet, Network, rpcNodeURL, d, bobWireAcc.Address(), ar, prBob)
	setup.ChannelServiceClients = []proto.ChannelServiceClient{aliceCSClient, bobCSClient}
	setup.ChannelServiceCleanupFuncs = []func(){aliceCSCleanup, bobCSCleanup}
	log.Printf("Participants: %v", parts)

	// Initialize Users
	for i, part := range parts {
		if i == 0 {
			_, err = aliceCS.InitializeUser(part, aliceWSC, external.NewWallet(chwallet.NewExternalClient(aliceWSC)))
		} else {
			_, err = bobCS.InitializeUser(part, bobWSC, external.NewWallet(chwallet.NewExternalClient(bobWSC)))
		}
		require.NoError(t, err, "error initializing user %d", i)
	}

	setup.Asset = asset.Asset{
		IsCKBytes: true,
		SUDT:      nil,
	}
	return setup
}

func setupChannelService(t *testing.T, name string, wsc proto.WalletServiceClient, net *p2p.Net, network types.Network, rpcNodeUrl string, d backend.Deployment, wireAddr wire.Address, addrResolver service.AddressResolver, pr persistence.PersistRestorer) (proto.ChannelServiceClient, *service.ChannelService, func()) {
	cs, err := service.NewChannelService(wsc, net, network, rpcNodeUrl, d, wireAddr, addrResolver, pr, name)
	require.NoError(t, err, "error setting up channel service for %s", name)
	lis := bufconn.Listen(bufSize)
	baseServer := grpc.NewServer()
	log.Printf("Registering channel service server for %s", name)
	proto.RegisterChannelServiceServer(baseServer, cs)
	go func() {
		err := baseServer.Serve(lis)
		require.NoError(t, err, "Server exited with error for %s", name)
	}()
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (defaultnet.Conn, error) {
		return lis.Dial()

	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet for %s", name)

	return proto.NewChannelServiceClient(conn), cs, func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}
}

func (set *Setup) setupWalletService(t *testing.T, name string, ctx context.Context, account *ckbwallet.Account, privateKey *secp256k1.PrivateKey, network types.Network) (proto.WalletServiceClient, func()) {
	lis := bufconn.Listen(bufSize)
	wsc := test.NewWalletServiceServer(name, account, privateKey, network)
	set.WalletServices = append(set.WalletServices, wsc)
	baseServer := grpc.NewServer()
	proto.RegisterWalletServiceServer(baseServer, wsc)
	go func() {
		err := baseServer.Serve(lis)
		require.NoError(t, err, "Server exited with error for %s", name)
	}()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (defaultnet.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet for %s", name)

	return proto.NewWalletServiceClient(conn), func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}
}

func MakeParticipants(pks []*secp256k1.PublicKey) ([]ckbaddr.Participant, error) {
	parts := make([]ckbaddr.Participant, len(pks))
	for i := range pks {
		part, err := ckbaddr.NewDefaultParticipant(pks[i])
		if err != nil {
			return nil, fmt.Errorf("unable to create participant: %w", err)
		}
		parts[i] = *part
	}
	return parts, nil
}

func parseSUDTOwnerLockArg(pathToSUDTOwnerLockArg string) (string, error) {
	b, err := os.ReadFile(pathToSUDTOwnerLockArg)
	if err != nil {
		return "", fmt.Errorf("reading sudt owner lock arg from file: %w", err)
	}
	sudtOwnerLockArg := string(b)
	if sudtOwnerLockArg == "" {
		return "", errors.New("sudt owner lock arg not found in file")
	}
	return sudtOwnerLockArg, nil
}
