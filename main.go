package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"perun.network/channel-service/deployment"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/channel-service/wallet"
	"perun.network/go-perun/wire"
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
	host := flag.String("host", "", "Where to host Channel Service Server, e.g. localhost:4321")
	aliceWssURL := flag.String("alice-wss-url", "", "URL of the WalletServiceServer e.g. localhost:1234")
	bobWssURL := flag.String("bob-wss-url", "", "URL of the WalletServiceServer e.g. localhost:1234")

	flag.Parse()

	// Check if the node URL is provided
	if *nodeURL == "" || *host == "" || *aliceWssURL == "" || *bobWssURL == "" {
		fmt.Printf("Usage:\n%s -node-url <node_url> -host <host_url> -alice-wss-url <wallet_service_url> -bob-wss-url <wallet_service_url> [public_key1] [public_key2] ...\n", filepath.Base(os.Args[0]))
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
	bus := wire.NewLocalBus()
	cs, err := service.NewChannelService(nil, bus, types.NetworkTest, *nodeURL, d)
	if err != nil {
		log.Fatalf("error setting up channel service: %v", err)
	}

	log.Printf("Participants: %v", parts)
	// Initialize Users
	for i, part := range parts {
		if i == 0 {
			_, err = cs.InitializeUser(part, aliceWSC, external.NewWallet(wallet.NewExternalClient(aliceWSC)))
		} else {
			_, err = cs.InitializeUser(part, bobWSC, external.NewWallet(wallet.NewExternalClient(bobWSC)))
		}
		if err != nil {
			log.Fatalf("error initializing user: %v", err)
		}
	}

	// Set up ChannelService Server
	lis, err := net.Listen("tcp", *host)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterChannelServiceServer(grpcServer, cs)
	err = grpcServer.Serve(lis)
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
