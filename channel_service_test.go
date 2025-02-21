package main

import (
	"context"
	"log"
	"time"

	"perun.network/channel-service/rpc/proto"

	"testing"

	"github.com/stretchr/testify/require"
	"perun.network/channel-service/test"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wire/protobuf"
)

func TestRestoreChannels(t *testing.T) {
	t.Run("payment-channels on ckb example", func(t *testing.T) {
		runRestoreChannels(t)
	})
}

func runRestoreChannels(t *testing.T) {
	setup := test.NewTestSetup(t)
	defer setup.WscCleanupFuncs[0]()
	defer setup.WscCleanupFuncs[1]()

	aliceChannelServiceClient := setup.ChannelServiceClients[0]
	bobChannelServiceClient := setup.ChannelServiceClients[1]

	// Bob and alice accept all incoming changes.
	aliceWalletService := setup.WalletServices[0]
	bobWalletService := setup.WalletServices[1]

	aliceWalletService.SetOpenChannelResponse(true)
	aliceWalletService.SetSignMessageResponse(true)
	aliceWalletService.SetSignTransactionResponse(true)

	bobWalletService.SetOpenChannelResponse(true)
	bobWalletService.SetSignMessageResponse(true)
	bobWalletService.SetSignTransactionResponse(true)

	ckbAsset := setup.Asset
	assetsmap := map[channel.Asset]float64{
		&ckbAsset: 200.0,
	}

	// Open channel.
	aliceChannelOpenRequest, err := test.NewChannelOpenRequest(setup.Participants[0], setup.Participants[1], assetsmap)
	require.NoError(t, err)

	openChannelResp, err := aliceChannelServiceClient.OpenChannel(context.Background(), &aliceChannelOpenRequest)
	log.Println("Channel Opened")
	require.NoError(t, err)
	require.NotNil(t, openChannelResp)

	acp, ok := openChannelResp.Msg.(*proto.ChannelOpenResponse_ChannelId)
	require.True(t, ok)
	require.NotNil(t, acp)

	// Wait for channel to be funded.
	log.Println("Waiting for channel to be funded")
	time.Sleep(1 * time.Second)

	// Close perun clients
	log.Println("Closing Alice's client")
	_, err = aliceChannelServiceClient.ClosePerunClient(context.TODO(), &proto.ClosePerunClientRequest{})
	require.NoError(t, err)
	log.Println("Closing bob's client")
	_, err = bobChannelServiceClient.ClosePerunClient(context.TODO(), &proto.ClosePerunClientRequest{})
	require.NoError(t, err)

	// Restart Perun clients
	log.Println("Restarting Alice's client")
	_, err = aliceChannelServiceClient.NewPerunClient(context.TODO(), test.NewPerunClientRequest())
	require.NoError(t, err)
	log.Println("Restarting Bob's client")
	_, err = bobChannelServiceClient.NewPerunClient(context.TODO(), test.NewPerunClientRequest())

	// Restore channels
	log.Println("Restoring channels for Alice")
	_, err = aliceChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)
	log.Println("Restoring channels for Bob")
	_, err = bobChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)

	// Perform update on restored channels
	log.Println("Performing update on restored channels")
	updateState, err := protobuf.ToState(aliceWalletService.CurrentState)
	require.NoError(t, err)
	require.NotNil(t, updateState)

	ammounts := map[channel.Asset]float64{
		&ckbAsset: 20.0,
	}

	aliceChannelUpdateRequest, err := test.NewChannelUpdateRequest([32]byte(acp.ChannelId), updateState, ammounts, 0)
	require.NoError(t, err)
	require.NotNil(t, aliceChannelUpdateRequest)

	updateChannelResp, err := aliceChannelServiceClient.UpdateChannel(context.TODO(), aliceChannelUpdateRequest)
	require.NoError(t, err)
	require.NotNil(t, updateChannelResp)
	updateAcp, ok := updateChannelResp.Msg.(*proto.ChannelUpdateResponse_Update)
	require.True(t, ok)
	require.Equal(t, updateAcp.Update.ChannelId, acp.ChannelId)

	// Restart Channel Services to check if the restored channels are still there
	log.Println("Closing Channel Services")
	setup.ChannelServiceCleanupFuncs[0]()
	setup.ChannelServiceCleanupFuncs[1]()

	setup.RestartChannelServices(t)
	aliceChannelServiceClient = setup.ChannelServiceClients[0]
	bobChannelServiceClient = setup.ChannelServiceClients[1]

	log.Println("Restoring channels from database")
	_, err = aliceChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)
	_, err = bobChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)

	// Close the channel
	log.Println("Closing the channel")
	closeChannelResponse, err := aliceChannelServiceClient.CloseChannel(context.TODO(), test.NewChannelCloseRequest([32]byte(acp.ChannelId)))
	require.NoError(t, err)
	require.NotNil(t, closeChannelResponse)
	closeAcp, ok := closeChannelResponse.Msg.(*proto.ChannelCloseResponse_Close)
	require.True(t, ok)
	require.Equal(t, closeAcp.Close.ChannelId, acp.ChannelId)

	log.Println("Test run sucessfully")
	setup.ChannelServiceCleanupFuncs[0]()
	setup.ChannelServiceCleanupFuncs[1]()
}
