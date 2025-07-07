package main

import (
	"context"
	"log"
	"time"

	"perun.network/channel-service/rpc/proto"

	"testing"

	"github.com/stretchr/testify/require"
	chanserv "perun.network/channel-service/service"
	"perun.network/channel-service/test"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wire/protobuf"
)

func TestRestoreChannels(t *testing.T) {
	t.Run("payment-channels on ckb example", func(t *testing.T) {
		runRestoreChannels(t)
	})
}

func TestGetChannels(t *testing.T) {
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
		&ckbAsset: 100.0,
	}

	// Alice opens channel Open channel.
	aliceChannelOpenRequest, err := test.NewChannelOpenRequest(setup.Participants[0], setup.Participants[1], assetsmap)
	require.NoError(t, err)

	openChannelResp, err := aliceChannelServiceClient.OpenChannel(context.Background(), &aliceChannelOpenRequest)
	log.Println("Channel Opened")
	require.NoError(t, err)
	require.NotNil(t, openChannelResp)

	acp, ok := openChannelResp.Msg.(*proto.ChannelOpenResponse_ChannelId)
	require.True(t, ok)
	require.NotNil(t, acp)
	firstChannelID := acp.ChannelId

	// Wait for channel to be funded.
	log.Println("Waiting for channel to be funded")
	time.Sleep(5 * time.Second)

	// Bob opens another channel
	bobChannelOpenRequest, err := test.NewChannelOpenRequest(setup.Participants[1], setup.Participants[0], assetsmap)
	require.NoError(t, err)
	openChannelResp, err = bobChannelServiceClient.OpenChannel(context.Background(), &bobChannelOpenRequest)
	log.Println("Channel Opened by Bob")
	require.NoError(t, err)
	require.NotNil(t, openChannelResp)
	bcp, ok := openChannelResp.Msg.(*proto.ChannelOpenResponse_ChannelId)
	require.True(t, ok)
	require.NotNil(t, bcp)
	secondChannelID := bcp.ChannelId
	// Wait for channel to be funded.
	log.Println("Waiting for channel to be funded by Bob")
	time.Sleep(5 * time.Second)

	// Prep for getChannels request
	aliceParticipant := setup.Participants[0]
	bobParticipant := setup.Participants[1]
	aliceParticipantBytes, err := aliceParticipant.MarshalBinary()
	require.NoError(t, err)
	bobParticipantBytes, err := bobParticipant.MarshalBinary()
	require.NoError(t, err)
	assertChannelStatesType := func(t *testing.T, resp *proto.GetChannelsResponse) *proto.ChannelStates {
		require.NotNil(t, resp, "response should not be nil")
		require.NotNil(t, resp.GetChannelStates().GetStates(), "ChannelStates field should not be nil")
		return resp.GetChannelStates()
	}
	// Get channels for Alice
	aliceGetChannelsResp, err := aliceChannelServiceClient.GetChannels(context.Background(), &proto.GetChannelsRequest{Requester: aliceParticipantBytes})
	require.NoError(t, err)
	assertChannelStatesType(t, aliceGetChannelsResp)
	require.Len(t, aliceGetChannelsResp.GetChannelStates().GetStates(), len(aliceGetChannelsResp.GetChannelStates().GetActorIdxs()), "Alice should have the same number of channel states and actor indexes")

	// Get channels for Bob
	bobGetChannelsResp, err := bobChannelServiceClient.GetChannels(context.Background(), &proto.GetChannelsRequest{Requester: bobParticipantBytes})
	require.NoError(t, err)
	assertChannelStatesType(t, bobGetChannelsResp)
	require.Len(t, bobGetChannelsResp.GetChannelStates().GetStates(), len(bobGetChannelsResp.GetChannelStates().GetActorIdxs()), "Bob should have the same number of channel states and actor indexes")

	// Create a set of expected channel IDs
	expectedChannelIDs := make(map[channel.ID]string)
	expectedChannelIDs[channel.ID(firstChannelID)] = "alice" // Alice proposed the first channel
	expectedChannelIDs[channel.ID(secondChannelID)] = "bob"  // Bob proposed the second channel

	// Create sets for Alice's and Bob's channel IDs
	aliceChannelIDs := make(map[channel.ID]bool)
	aliceActorIndexes := make(map[channel.ID]uint32)
	bobChannelIDs := make(map[channel.ID]bool)
	bobActorIndexes := make(map[channel.ID]uint32)

	// Populate Alice's channel IDs
	for i, pState := range aliceGetChannelsResp.GetChannelStates().GetStates() {
		aliceState, err := chanserv.AsChannelState(pState)
		require.NoError(t, err, "Alice's channel state should be valid")
		aliceChannelIDs[aliceState.ID] = true
		aliceActorIndexes[aliceState.ID] = aliceGetChannelsResp.GetChannelStates().GetActorIdxs()[i]
	}

	// Populate Bob's channel IDs
	for i, pState := range bobGetChannelsResp.GetChannelStates().GetStates() {
		bobState, err := chanserv.AsChannelState(pState)
		require.NoError(t, err, "Bob's channel state should be valid")
		bobChannelIDs[bobState.ID] = true
		bobActorIndexes[bobState.ID] = bobGetChannelsResp.GetChannelStates().GetActorIdxs()[i]
	}

	// Check if both Alice and Bob have all expected channel IDs
	for expectedID := range expectedChannelIDs {
		require.True(t, aliceChannelIDs[expectedID], "Alice should have channel ID %v", expectedID)
		require.True(t, bobChannelIDs[expectedID], "Bob should have channel ID %v", expectedID)
		log.Printf("Both Alice and Bob have channel ID %v", expectedID)
	}

	// check if actor_indexes are correct for both participants
	for expectedID, proposer := range expectedChannelIDs {
		aliceIdx := aliceActorIndexes[expectedID]
		bobIdx := bobActorIndexes[expectedID]
		if proposer == "alice" {
			require.Equal(t, uint32(0), aliceIdx, "Alice should be the proposer for channel ID %v", expectedID)
			require.Equal(t, uint32(1), bobIdx, "Bob should be the responder for channel ID %v", expectedID)
		}
		if proposer == "bob" {
			require.Equal(t, uint32(1), aliceIdx, "Alice should be the responder for channel ID %v", expectedID)
			require.Equal(t, uint32(0), bobIdx, "Bob should be the proposer for channel ID %v", expectedID)
		}
		log.Printf("Channel ID %v has correct actor indexes: Alice(%d), Bob(%d)", expectedID, aliceIdx, bobIdx)
	}

	// Check if the number of channels matches the expected count
	require.Len(t, aliceChannelIDs, len(expectedChannelIDs), "Alice should have exactly %d channels", len(expectedChannelIDs))
	require.Len(t, bobChannelIDs, len(expectedChannelIDs), "Bob should have exactly %d channels", len(expectedChannelIDs))

	log.Println("Alice and Bob have the same channels")
	// Close the channels
	log.Println("Closing the channel opened by Alice")
	closeChannelResponse, err := aliceChannelServiceClient.CloseChannel(context.Background(), test.NewChannelCloseRequest([32]byte(firstChannelID)))
	require.NoError(t, err)
	require.NotNil(t, closeChannelResponse)
	closeAcp, ok := closeChannelResponse.Msg.(*proto.ChannelCloseResponse_Close)
	require.True(t, ok)
	require.Equal(t, closeAcp.Close.ChannelId, firstChannelID)

	log.Println("Closing the channel opened by Bob")
	closeChannelResponse, err = bobChannelServiceClient.CloseChannel(context.Background(), test.NewChannelCloseRequest([32]byte(secondChannelID)))
	require.NoError(t, err)
	require.NotNil(t, closeChannelResponse)
	closeBcp, ok := closeChannelResponse.Msg.(*proto.ChannelCloseResponse_Close)
	require.True(t, ok)
	require.Equal(t, closeBcp.Close.ChannelId, secondChannelID)

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
	sudtAsset := setup.SudtAsset
	assetsmap := map[channel.Asset]float64{
		&ckbAsset:  100.0,
		&sudtAsset: 1.0,
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
	require.NoError(t, err)

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
		&ckbAsset:  20.0,
		&sudtAsset: 10.0,
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

	// Close perun clients
	log.Println("Closing Alice's client")
	_, err = aliceChannelServiceClient.ClosePerunClient(context.TODO(), &proto.ClosePerunClientRequest{})
	require.NoError(t, err)
	log.Println("Closing bob's client")
	_, err = bobChannelServiceClient.ClosePerunClient(context.TODO(), &proto.ClosePerunClientRequest{})
	require.NoError(t, err)

	log.Println("Test run sucessfully")
	setup.ChannelServiceCleanupFuncs[0]()
	setup.ChannelServiceCleanupFuncs[1]()
}
