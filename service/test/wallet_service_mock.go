package test

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wire/protobuf"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/perun-ckb-backend/backend"
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

	CurrentState *protobuf.State

	proto.UnimplementedWalletServiceServer
}

func NewWalletServiceServer(name string, acc *wallet.Account, privKey *secp256k1.PrivateKey, network types.Network) *MyWalletService {
	return &MyWalletService{
		name:       name,
		account:    acc,
		privateKey: privKey,
		network:    network,
	}
}
func (wsc *MyWalletService) OpenChannel(ctx context.Context, in *proto.OpenChannelRequest) (*proto.OpenChannelResponse, error) {
	if wsc.openChannelResponseFlag {
		return openChannelAccepted()
	} else {
		return openChannelRejected()
	}
}

// SetOpenChannelResponse sets default for wallet's response for incoming channel requests. True sets to accept all proposals. False sets it to reject
func (wsc *MyWalletService) SetOpenChannelResponse(flag bool) {
	wsc.openChannelResponseFlag = flag
}

func openChannelAccepted() (*proto.OpenChannelResponse, error) {
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
		return wsc.signMessageAccepted(in.Data)
	} else {
		return wsc.signMessageRejected()
	}
}

// SetSignMessageResponse set default response for SignMessage. True sets wallet to sign all messages, false rejects all requests to sign message
func (wsc *MyWalletService) SetSignMessageResponse(flag bool) {
	wsc.signMessageResponseFlag = flag
}

func (wsc *MyWalletService) signMessageAccepted(data []byte) (*proto.SignMessageResponse, error) {
	//How to sign message
	signedMsg, err := wsc.account.SignData(data)
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
