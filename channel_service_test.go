package main

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/go-perun/wire/protobuf"

	//"log"
	//"net"
	"testing"

	"github.com/stretchr/testify/require"
	"perun.network/channel-service/test"
	"perun.network/go-perun/channel"
	//ckbasset "perun.network/perun-ckb-backend/channel/asset"
)

func TestRestoreChannels(t *testing.T) {
	t.Run("payment-channels on ckb example", func(t *testing.T) {
		runRestoreChannels(t)
	})
}

func runRestoreChannels(t *testing.T) {
	setup := test.NewTestSetup(t)
	//aliceWalletServiceClient := setup.WalletServiceClients[0]
	defer setup.WscCleanupFuncs[0]()
	//bobWalletServiceClient := setup.WalletServiceClients[1]
	defer setup.WscCleanupFuncs[1]()

	aliceChannelServiceClient := setup.ChannelServiceClients[0]
	defer setup.ChannelServiceCleanupFuncs[0]()
	bobChannelServiceClient := setup.ChannelServiceClients[1]
	defer setup.ChannelServiceCleanupFuncs[1]()

	// set so that bob and alice accept all incoming changes
	aliceWalletService := setup.WalletServices[0]
	bobWalletService := setup.WalletServices[1]

	aliceWalletService.SetOpenChannelResponse(true)
	aliceWalletService.SetSignMessageResponse(true)
	aliceWalletService.SetSignTransactionResponse(true)
	aliceWalletService.SetUpdateNotificationResponse(true)
	//aliceWalletService.SetUpdateNotificationCounter(3)

	bobWalletService.SetOpenChannelResponse(true)
	bobWalletService.SetSignMessageResponse(true)
	bobWalletService.SetSignTransactionResponse(true)
	bobWalletService.SetUpdateNotificationResponse(true)
	//bobWalletService.SetUpdateNotificationCounter(3)

	ckbAsset := setup.Asset
	assetsmap := map[channel.Asset]float64{
		&ckbAsset: 100.0,
	}

	// open channel
	aliceChannelOpenRequest, err := test.NewChannelOpenRequest(setup.Participants[0], setup.Participants[1], assetsmap)
	require.NoError(t, err)

	openChannelResp, err := aliceChannelServiceClient.OpenChannel(context.Background(), &aliceChannelOpenRequest)
	require.NoError(t, err)
	require.NotNil(t, openChannelResp)
	log.Println("Channel Opened")
	log.Println("")

	//update channel
	log.Println("Alice sends 10CkBytes to Bob")
	channelId, err := service.AsChannelID(openChannelResp.GetChannelId())
	require.NoError(t, err)

	aliceParticipantInBytes, err := setup.Participants[0].MarshalBinary()
	require.NoError(t, err)
	getChannelResp, err := aliceChannelServiceClient.GetChannels(context.Background(), test.GetChannelsRequest(aliceParticipantInBytes))
	require.NoError(t, err)
	channelState, err := service.AsChannelState(getChannelResp.GetState())
	require.NoError(t, err)
	log.Print("Allocation before update ")
	gpAlloc, err := protobuf.ToAllocation(getChannelResp.GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	channelUpdateReq, err := test.NewChannelUpdateRequest(channelId, channelState, map[channel.Asset]float64{
		&ckbAsset: 10.0,
	}, 0)
	require.NoError(t, err)

	updateChannelResp, err := aliceChannelServiceClient.UpdateChannel(context.Background(), channelUpdateReq)
	require.NoError(t, err)
	gpAlloc, err = protobuf.ToAllocation(updateChannelResp.GetUpdate().GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	log.Print("Alice successfully sent 10 CKbytes to Bob\n\n")

	//aliceSendToBob(t, setup, 10.0)
	//bobSendToAlice(t, setup, 10.0)

	// close perun clients w/o closing channels
	log.Println("Closing Alice's client")
	_, err = aliceChannelServiceClient.ClosePerunClient(context.TODO(), &emptypb.Empty{})
	require.NoError(t, err)
	log.Println("Closing bob's client")
	_, err = bobChannelServiceClient.ClosePerunClient(context.TODO(), &emptypb.Empty{})
	require.NoError(t, err)

	// restart Perun clients
	log.Println("Restarting Alice's client")
	_, err = aliceChannelServiceClient.NewPerunClient(context.TODO(), test.NewPerunClientRequest())
	require.NoError(t, err)
	log.Println("Restarting Bob's client")
	_, err = bobChannelServiceClient.NewPerunClient(context.TODO(), test.NewPerunClientRequest())

	// restore channels
	log.Println("Restoring channels for Alice")
	_, err = aliceChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)
	log.Println("Restoring channels for Bob")
	_, err = bobChannelServiceClient.RestoreChannels(context.TODO(), &proto.RestoreChannelsRequest{})
	require.NoError(t, err)

	// check balances after restoring channels
	getChannelResp, err = aliceChannelServiceClient.GetChannels(context.Background(), test.GetChannelsRequest(aliceParticipantInBytes))
	require.NoError(t, err)
	channelState, err = service.AsChannelState(getChannelResp.GetState())
	require.NoError(t, err)
	log.Println("Allocation after restoring channels")
	gpAlloc, err = protobuf.ToAllocation(getChannelResp.GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))

	//bobSendToAlice(t, setup, 20.0)
	aliceSendToBob(t, setup, 20.0)
	log.Println("Test run sucessfully")

}

func aliceSendToBob(t *testing.T, setup *test.Setup, amount float64) {
	log.Println("Alice sends ", amount, "CkBytes to Bob")
	aliceParticipantInBytes, err := setup.Participants[0].MarshalBinary()
	require.NoError(t, err)
	getChannelResp, err := setup.ChannelServiceClients[0].GetChannels(context.Background(), test.GetChannelsRequest(aliceParticipantInBytes))
	require.NoError(t, err)
	channelState, err := service.AsChannelState(getChannelResp.GetState())
	require.NoError(t, err)
	log.Print("Allocation before update")
	gpAlloc, err := protobuf.ToAllocation(getChannelResp.GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	channelUpdateReq, err := test.NewChannelUpdateRequest(channelState.ID, channelState, map[channel.Asset]float64{
		&setup.Asset: amount,
	}, 0)
	require.NoError(t, err)

	updateChannelResp, err := setup.ChannelServiceClients[0].UpdateChannel(context.Background(), channelUpdateReq)
	require.NoError(t, err)
	gpAlloc, err = protobuf.ToAllocation(updateChannelResp.GetUpdate().GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	log.Print("Alice successfully sent ", amount, "CKbytes to Bob\n\n")
}

func bobSendToAlice(t *testing.T, setup *test.Setup, amount float64) {
	log.Println("Bob sends ", amount, "CkBytes to Alice")
	bobParticipantInBytes, err := setup.Participants[1].MarshalBinary()
	require.NoError(t, err)
	getChannelResp, err := setup.ChannelServiceClients[1].GetChannels(context.Background(), test.GetChannelsRequest(bobParticipantInBytes))
	require.NoError(t, err)
	channelState, err := service.AsChannelState(getChannelResp.GetState())
	require.NoError(t, err)
	log.Print("Allocation before update")
	gpAlloc, err := protobuf.ToAllocation(getChannelResp.GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	channelUpdateReq, err := test.NewChannelUpdateRequest(channelState.ID, channelState, map[channel.Asset]float64{
		&setup.Asset: amount,
	}, 1)
	require.NoError(t, err)

	updateChannelResp, err := setup.ChannelServiceClients[1].UpdateChannel(context.Background(), channelUpdateReq)
	require.NoError(t, err)
	gpAlloc, err = protobuf.ToAllocation(updateChannelResp.GetUpdate().GetState().GetAllocation())
	require.NoError(t, err)
	log.Println(test.AllocToString(gpAlloc))
	log.Print("Bob successfully sent ", amount, "CKbytes to Alice\n\n")
}
