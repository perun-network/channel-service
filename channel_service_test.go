package main

import (
	"context"
	//"log"
	//"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/test/bufconn"

	"perun.network/channel-service/test"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func TestRestoreChannels(t *testing.T) {
	t.Run("payment-channels on ckb example", func(t *testing.T) {
		runRestoreChannels(t)
	})
}

func runRestoreChannels(t *testing.T) {
	setup := test.NewTestSetup(t)
	aliceWalletServiceClient := setup.WalletServiceClients[0]
	defer setup.WscCleanupFuncs[0]()
	bobWalletServiceClient := setup.WalletServiceClients[1]
	defer setup.WscCleanupFuncs[1]()

	aliceChannelServiceClient := setup.ChannelServiceClients[0]
	defer setup.ChannelServiceCleanupFuncs[0]()
	bobChannelServiceClient := setup.ChannelServiceClients[1]
	defer setup.ChannelServiceCleanupFuncs[1]()

	// set so that bob and alice accept all incoming changes
	aliceWalletService := setup.WalletServices[0]
	bobWalletService := setup.WalletServices[1]
	
	aliceWalletService.SetOpenChannelResponse(true)
	bobWalletService.SetOpenChannelResponse(true)

	// open channel
	aliceOpenChannelReq := test.NewOpenChannelRequest()
	openChannelResp, err := aliceChannelServiceClient.OpenChannel(context.Background(), aliceOpenChannelReq)
	require.NoError(t, err)
	require.NotNil(t, openChannelResp)

	
	/*
	aliceWalletService.SetOpenChannelResponse(true)
	bobChannelService.SetOpenChannelResponse(true)

	// open channel
	aliceOpenChannelReq := test.NewOpenChannelRequest()
	aliceOpenChannelResp, err := aliceWalletServiceClient.OpenChannel(context.Background(), aliceOpenChannelReq)
	require.NoError(t, err)
	require.NotNil(t, aliceOpenChannelResp)
	*/


}
