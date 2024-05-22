package test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"github.com/perun-network/perun-libp2p-wire/p2p"
	"github.com/stretchr/testify/require"
	"perun.network/channel-service/channel/client"
	"perun.network/channel-service/deployment"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/wallet"
	"polycry.pt/poly-go/sortedkv/memorydb"
)

const (
	rpcNodeURL = "http://localhost:8114"
	Network    = types.NetworkTest
	devNetDir  = "test/devnet"
)

type Setup struct {
	t                   *testing.T
	Deployment          backend.Deployment
	SUDTInfo            deployment.SUDTInfo
	WalletAccs          []*wallet.Account
	AccNets             []*p2p.Net
	AccPersistRestorers []persistence.PersistRestorer
	PaymenClients       []*client.PaymentClient
	Asset               asset.Asset
	AccKeys             []*secp256k1.PrivateKey
	EphemeralWallet     *wallet.EphemeralWallet
	WireAccs            []*p2p.Account
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

	w := wallet.NewEphemeralWallet()
	setup.EphemeralWallet = w

	keyAlice, err := deployment.GetKey(devNetDir + "/accounts/alice.pk")
	require.NoError(t, err, "error getting alice's private key")

	keyBob, err := deployment.GetKey(devNetDir + "/accounts/bob.pk")
	require.NoError(t, err, "error getting bob's private key")

	aliceAccount := wallet.NewAccountFromPrivateKey(keyAlice)
	bobAccount := wallet.NewAccountFromPrivateKey(keyBob)

	setup.WalletAccs = []*wallet.Account{aliceAccount, bobAccount}
	setup.AccKeys = []*secp256k1.PrivateKey{keyAlice, keyBob}

	err = w.AddAccount(aliceAccount)
	require.NoError(t, err, "error adding alice's account")

	err = w.AddAccount(bobAccount)
	require.NoError(t, err, "error adding bob's account")

	aliceWireAcc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	aliceNet, err := p2p.NewP2PBus(aliceWireAcc)
	require.NoError(t, err, "error creating p2p net")
	aliceBus := aliceNet.Bus
	aliceListener := aliceNet.Listener
	go aliceBus.Listen(aliceListener)

	bobWireAcc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	bobNet, err := p2p.NewP2PBus(bobWireAcc)
	require.NoError(t, err, "error creating p2p net")
	bobBus := bobNet.Bus
	bobListener := bobNet.Listener
	go bobBus.Listen(bobListener)

	setup.WireAccs = []*p2p.Account{aliceWireAcc, bobWireAcc}
	setup.AccNets = []*p2p.Net{aliceNet, bobNet}

	prAlice := keyvalue.NewPersistRestorer(memorydb.NewDatabase())
	prBob := keyvalue.NewPersistRestorer(memorydb.NewDatabase())
	setup.AccPersistRestorers = []persistence.PersistRestorer{prAlice, prBob}

	alice, err := client.NewPaymentClient(
		"Alice",
		Network,
		d,
		rpcNodeURL,
		aliceAccount,
		*keyAlice,
		w,
		prAlice,
		aliceWireAcc.Address(),
		aliceNet,
	)
	require.NoError(t, err, "error creating alice's client")

	bob, err := client.NewPaymentClient(
		"Bob",
		Network,
		d,
		rpcNodeURL,
		bobAccount,
		*keyBob,
		w,
		prBob,
		bobWireAcc.Address(),
		bobNet,
	)
	require.NoError(t, err, "error creating bob's client")

	setup.PaymenClients = []*client.PaymentClient{alice, bob}

	setup.Asset = asset.Asset{
		IsCKBytes: true,
		SUDT:      nil,
	}
	return setup
}

func (setup *Setup) NewPaymentClients(t *testing.T) {
	alice, err := client.NewPaymentClient(
		"Alice",
		Network,
		setup.Deployment,
		rpcNodeURL,
		setup.WalletAccs[0],
		*setup.AccKeys[0],
		setup.EphemeralWallet,
		setup.AccPersistRestorers[0],
		setup.WireAccs[0].Address(),
		setup.AccNets[0],
	)
	require.NoError(t, err, "failed to set up Alice's client")

	bob, err := client.NewPaymentClient(
		"Bob",
		Network,
		setup.Deployment,
		rpcNodeURL,
		setup.WalletAccs[1],
		*setup.AccKeys[1],
		setup.EphemeralWallet,
		setup.AccPersistRestorers[1],
		setup.WireAccs[1].Address(),
		setup.AccNets[1],
	)
	require.NoError(t, err, "failed to set up Bob's client")

	setup.PaymenClients = []*client.PaymentClient{alice, bob}
}

func parseSUDTOwnerLockArg(pathToSUDTOwnerLockArg string) (string, error) {
	b, err := ioutil.ReadFile(pathToSUDTOwnerLockArg)
	if err != nil {
		return "", fmt.Errorf("reading sudt owner lock arg from file: %w", err)
	}
	sudtOwnerLockArg := string(b)
	if sudtOwnerLockArg == "" {
		return "", errors.New("sudt owner lock arg not found in file")
	}
	return sudtOwnerLockArg, nil
}
