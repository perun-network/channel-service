package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"github.com/perun-network/perun-libp2p-wire/p2p"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"perun.network/channel-service/deployment"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/channel-service/wallet"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/wallet/external"
)

func SetLogFile(path string) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}

func main() {
	SetLogFile("demo.log")

	// Define command-line flags
	nodeURL := flag.String("node-url", "", "CKB node URL")
	hostA := flag.String("hostA", "", "Where to host Alice Channel Service Server, e.g. localhost:4321")
	hostB := flag.String("hostB", "", "Where to host Bob Channel Service Server, e.g. localhost:4321")
	aliceWssURL := flag.String("alice-wss-url", "", "URL of the WalletServiceServer e.g. localhost:1234")
	bobWssURL := flag.String("bob-wss-url", "", "URL of the WalletServiceServer e.g. localhost:1234")
	flag.Parse()

	// Check if the node URL is provided
	if *nodeURL == "" || *hostA == "" || *hostB == "" || *aliceWssURL == "" || *bobWssURL == "" {
		fmt.Printf("Usage:\n%s -node-url <node_url> -hostA <host_url> -hostB <host_url> -alice-wss-url <wallet_service_url> -bob-wss-url <wallet_service_url> [public_key1] [public_key2] ...\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	args := flag.Args()
	pubKeys := make([]secp256k1.PublicKey, len(args))
	// Iterate through the command-line arguments
	for i, arg := range flag.Args() {
		publicKeyStr := arg

		// Parse the public key
		publicKeyBytes, err := hex.DecodeString(publicKeyStr)
		if err != nil {
			log.Fatalf("error decoding public key: %v", err)
		}
		pubkey, err := secp256k1.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error parsing public key: %v", err)
		}
		pubKeys[i] = *pubkey
	}
	parts, err := MakeParticipants(pubKeys)
	if err != nil {
		log.Fatalf("error making participants: %v", err)
	}

	// Set up WalletService Client
	mkWSC := func(url string) proto.WalletServiceClient {
		conn, err := grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect to wallet service server: %v", err)
		}
		return proto.NewWalletServiceClient(conn)
	}

	aliceWSC := mkWSC(*aliceWssURL)
	bobWSC := mkWSC(*bobWssURL)

	// Set up ChannelService
	d, err := MakeDeployment()
	if err != nil {
		log.Fatalf("error getting deployment: %v", err)
	}

	wireAccA := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	netA, err := p2p.NewP2PBus(wireAccA)
	if err != nil {
		log.Fatalf("creating p2p net: %v", err)
	}
	go netA.Bus.Listen(netA.Listener)

	// AddressRessolver Alice
	arA := service.NewRelayServerResolver(wireAccA)

	csA, err := service.NewChannelService(nil, netA, types.NetworkTest, *nodeURL, d, wireAccA.Address(), arA)
	if err != nil {
		log.Fatalf("error setting up channel service: %v", err)
	}

	wireAccB := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	netB, err := p2p.NewP2PBus(wireAccB)
	if err != nil {
		log.Fatalf("creating p2p net: %v", err)
	}
	go netB.Bus.Listen(netB.Listener)

	// AddressRessolver Bob
	arB := service.NewRelayServerResolver(wireAccB)

	csB, err := service.NewChannelService(nil, netB, types.NetworkTest, *nodeURL, d, wireAccB.Address(), arB)
	if err != nil {
		log.Fatalf("error setting up channel service: %v", err)
	}

	log.Printf("Participants: %v", parts)
	// Initialize Users
	for i, part := range parts {
		if i == 0 {
			_, err = csA.InitializeUser(part, aliceWSC, external.NewWallet(wallet.NewExternalClient(aliceWSC)))
		} else {
			_, err = csB.InitializeUser(part, bobWSC, external.NewWallet(wallet.NewExternalClient(bobWSC)))
		}
		if err != nil {
			log.Fatalf("error initializing user: %v", err)
		}
	}

	// Set up ChannelService Server
	lisA, err := net.Listen("tcp", *hostA)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServerA := grpc.NewServer(opts...)
	proto.RegisterChannelServiceServer(grpcServerA, csA)
	err = grpcServerA.Serve(lisA)
	if err != nil {
		log.Fatal(err)
	}

	lisB, err := net.Listen("tcp", *hostB)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServerB := grpc.NewServer(opts...)
	proto.RegisterChannelServiceServer(grpcServerB, csB)
	err = grpcServerB.Serve(lisB)
	if err != nil {
		log.Fatal(err)
	}

}

func MakeParticipants(pks []secp256k1.PublicKey) ([]address.Participant, error) {
	parts := make([]address.Participant, len(pks))
	for i := range pks {
		part, err := address.NewDefaultParticipant(&pks[i])
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

func MakeDeployment() (backend.Deployment, error) {
	sudtOwnerLockArg, err := parseSUDTOwnerLockArg("./devnet/accounts/sudt-owner-lock-hash.txt")
	if err != nil {
		log.Fatalf("error getting SUDT owner lock arg: %v", err)
	}
	d, _, err := deployment.GetDeployment("./devnet/contracts/migrations/dev/", "./devnet/system_scripts", sudtOwnerLockArg)
	return d, err
}
