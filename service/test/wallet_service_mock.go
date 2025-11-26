package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wire/protobuf"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/perun-ckb-backend/backend"
	bchannel "perun.network/perun-ckb-backend/channel"
	"perun.network/perun-ckb-backend/wallet"
	"perun.network/perun-ckb-backend/wallet/address"
)

type MyWalletService struct {
	name                           string
	account                        *wallet.Account
	privateKey                     *secp256k1.PrivateKey
	network                        types.Network
	openChannelResponseFlag        bool
	updateNotificationResponseFlag bool
	signMessageResponseFlag        bool
	signTransactionResponseFlag    bool
	updateNotificationCounter      int
	TempChannelIDMap               map[string]bchannel.TempChannelID

	CurrentState *protobuf.State

	proto.UnimplementedWalletServiceServer
}

func NewWalletServiceServer(name string, acc *wallet.Account, privKey *secp256k1.PrivateKey, network types.Network) *MyWalletService {
	return &MyWalletService{
		name:             name,
		account:          acc,
		privateKey:       privKey,
		network:          network,
		TempChannelIDMap: make(map[string]bchannel.TempChannelID),
	}
}
func (wsc *MyWalletService) OpenChannel(ctx context.Context, in *proto.OpenChannelRequest) (*proto.OpenChannelResponse, error) {
	if wsc.openChannelResponseFlag {
		return wsc.openChannelAccepted(in)
	} else {
		return openChannelRejected()
	}
}

// SetOpenChannelResponse sets default for wallet's response for incoming channel requests. True sets to accept all proposals. False sets it to reject
func (wsc *MyWalletService) SetOpenChannelResponse(flag bool) {
	wsc.openChannelResponseFlag = flag
}

func (wsc *MyWalletService) openChannelAccepted(state *proto.OpenChannelRequest) (*proto.OpenChannelResponse, error) {
	// check if tempID field has been set and save it if it did
	data := state.GetProposal().GetBaseChannelProposal().GetInitData()

	if len(data) == 0 {
		log.Println("No tempID in proposal data")
	}
	tempID, err := bchannel.NewTempChannelIDFromBytes(data)
	if err != nil {
		log.Println("Error unmarshalling TempChannelID from proposal data", err)
	}
	if len(tempID) != bchannel.TempChannelIDLength {
		return nil, errors.New("invalid TempChannelID length, must be 32 bytes")
	}
	// wsc.TempChannelID = tempID
	wsc.TempChannelIDMap[tempID.String()] = tempID

	log.Println("TempChannelID set in wallet service for ", wsc.name, ":", wsc.TempChannelIDMap[tempID.String()])

	nonceShare := client.WithRandomNonce()["nonce"]
	// Check if nonceShare is of type [32]byte
	nonceShareArray, ok := nonceShare.([32]byte)
	if ok {
		// Convert the array to a slice
		nonceShareBytes := nonceShareArray[:]
		return &proto.OpenChannelResponse{
			Msg: &proto.OpenChannelResponse_NonceShare{
				NonceShare: nonceShareBytes,
			}}, nil
	}

	// If it's not [32]byte, check if it's already a byte slice
	nonceShareBytes, ok := nonceShare.([]byte)
	if !ok {
		log.Fatal("nonceShare is not a byte array")
	}

	return &proto.OpenChannelResponse{
		Msg: &proto.OpenChannelResponse_NonceShare{
			NonceShare: nonceShareBytes,
		}}, nil
}

func openChannelRejected() (*proto.OpenChannelResponse, error) {
	return &proto.OpenChannelResponse{
		Msg: &proto.OpenChannelResponse_Rejected{
			Rejected: &proto.Rejected{
				Reason: "not accepting channels",
			},
		},
	}, nil
}

func (wsc *MyWalletService) UpdateNotification(ctx context.Context, in *proto.UpdateNotificationRequest) (*proto.UpdateNotificationResponse, error) {
	wsc.CurrentState = in.State
	return updateNotificationAccepted()
}

// SetUpdateNotificationResponse set default response for UpdateNotification. True sets wallet to accep all channel updates. False rejects all channel updates
func (wsc *MyWalletService) SetUpdateNotificationResponse(flag bool) {
	wsc.updateNotificationResponseFlag = flag
}

func updateNotificationAccepted() (*proto.UpdateNotificationResponse, error) {
	return &proto.UpdateNotificationResponse{
		Accepted: true,
	}, nil
}

func updateNotificationRejected() (*proto.UpdateNotificationResponse, error) {
	return &proto.UpdateNotificationResponse{
		Accepted: false,
	}, nil
}

func (wsc *MyWalletService) SignMessage(ctx context.Context, in *proto.SignMessageRequest) (*proto.SignMessageResponse, error) {
	if wsc.signMessageResponseFlag {
		return wsc.signMessageAccepted(in)
	} else {
		return wsc.signMessageRejected()
	}
}

// SetSignMessageResponse set default response for SignMessage. True sets wallet to sign all messages, false rejects all requests to sign message
func (wsc *MyWalletService) SetSignMessageResponse(flag bool) {
	wsc.signMessageResponseFlag = flag
}

func (wsc *MyWalletService) signMessageAccepted(data *proto.SignMessageRequest) (*proto.SignMessageResponse, error) {
	tempIDBinary := data.GetTempChannelID()
	tempID, err := bchannel.NewTempChannelIDFromBytes(tempIDBinary)
	if err != nil {
		log.Println("Error converting data to TempChannelID", err)
		return nil, err
	}
	log.Println("TempChannelID in request:", tempID, "for user", wsc.name)
	log.Println("TempChannelID in wallet service:", wsc.TempChannelIDMap[tempID.String()], "for user", wsc.name)
	storedTempID := wsc.TempChannelIDMap[tempID.String()]
	if !bytes.Equal(storedTempID[:], tempID[:]) {
		return nil, errors.New("TempChannelID in request does not match the one stored in wallet service")
	}
	data1 := data.GetData()
	if len(data1) == 0 {
		return nil, errors.New("data to be signed is empty")
	}
	signedMsg, err := wsc.account.SignData(append(tempID[:], data1...))
	if err != nil {
		log.Println("Error signing message", err)
	}
	return &proto.SignMessageResponse{
		Msg: &proto.SignMessageResponse_Signature{
			Signature: signedMsg,
		},
	}, nil
}

func (wsc *MyWalletService) signMessageRejected() (*proto.SignMessageResponse, error) {
	return &proto.SignMessageResponse{
		Msg: &proto.SignMessageResponse_Rejected{
			Rejected: &proto.Rejected{
				Reason: "Not accepting messages",
			},
		}}, nil
}

func (wsc *MyWalletService) SignTransaction(ctx context.Context, tx *proto.SignTransactionRequest) (*proto.SignTransactionResponse, error) {
	if wsc.signTransactionResponseFlag {
		return wsc.signedTransactionAccepted(tx)
	} else {
		return wsc.signedTransationRejected()
	}
}

// SetSignTransactionResponse set default response for SignTransaction. True sets wallet to sign all tx requests, and false to reject all tx requests
func (wsc *MyWalletService) SetSignTransactionResponse(flag bool) {
	wsc.signTransactionResponseFlag = flag
}

func (wsc *MyWalletService) signedTransactionAccepted(tx *proto.SignTransactionRequest) (*proto.SignTransactionResponse, error) {
	ckbAddr := address.AsParticipant(wsc.account.Address()).ToCKBAddress(wsc.network)
	txSigner := backend.NewSignerInstance(ckbAddr, *wsc.privateKey, types.NetworkTest)
	//wrappedTx := &utils.TransactionWithScriptGroupsWrapper{}
	txWithScriptGroups := transaction.TransactionWithScriptGroups{}
	err := json.Unmarshal(tx.Transaction, &txWithScriptGroups)
	if err != nil {
		log.Println("Error unmarshalling transaction", err)
	}
	signedTx, err := txSigner.SignTransaction(&txWithScriptGroups)
	if err != nil {
		log.Println("Error signing transaction", err)
	}
	signedTxBytes, err := json.Marshal(signedTx)
	if err != nil {
		log.Println("Error marshalling signed transaction", err)
	}
	log.Println("SUCCESS: " + wsc.name + " successfully signed transaction")
	return &proto.SignTransactionResponse{
		Msg: &proto.SignTransactionResponse_Transaction{
			Transaction: signedTxBytes,
		},
	}, nil
}

func (wsc *MyWalletService) signedTransationRejected() (*proto.SignTransactionResponse, error) {
	return &proto.SignTransactionResponse{
		Msg: &proto.SignTransactionResponse_Rejected{
			Rejected: &proto.Rejected{
				Reason: "Not accepting transactions",
			},
		},
	}, nil

}

// SetGetAssetsResponse sets default response for GetAssets().
func (wsc *MyWalletService) SetGetAssetsResponse(flag bool) {}

func (wsc *MyWalletService) GetAssets(ctx context.Context, in *proto.GetAssetsRequest) (*proto.GetAssetsResponse, error) {
	return nil, nil
}

func (wsc *MyWalletService) mustEmbedUnimplementedWalletServiceServer() {}
