package test

import (
	"context"
	"encoding/json"
	"log"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"google.golang.org/grpc"
	"perun.network/channel-service/rpc/proto"
	"perun.network/go-perun/client"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/wallet"
	"perun.network/perun-ckb-backend/wallet/address"
)

// test implementation for wallet API
type MyWalletServiceClient struct {
	account                        *wallet.Account
	privateKey                     *secp256k1.PrivateKey
	network                        types.Network
	openChannelResponseFlag        bool
	updateNotificationResponseFlag bool
	signMessageResponseFlag        bool
	signTransactionResponseFlag    bool
}

func (wsc *MyWalletServiceClient) OpenChannel(ctx context.Context, in *proto.OpenChannelRequest, opts ...grpc.CallOption) (*proto.OpenChannelResponse, error) {
	if wsc.openChannelResponseFlag {
		return openChannelAccepted()
	} else {
		return openChannelRejected()
	}
}

// sets default for wallet's response for incoming channel reqeuests. True sets to accept all proposals. False sets it to reject
func (wsc *MyWalletServiceClient) SetOpenChannelResponse(flag bool) {
	wsc.openChannelResponseFlag = flag
}

func openChannelAccepted() (*proto.OpenChannelResponse, error) {
	nonceShare := client.WithRandomNonce()["nonce"]
	return &proto.OpenChannelResponse{
		Msg: &proto.OpenChannelResponse_NonceShare{
			NonceShare: nonceShare.([]byte),
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

func (wsc *MyWalletServiceClient) UpdateNotification(ctx context.Context, in *proto.UpdateNotificationRequest, opts ...grpc.CallOption) (*proto.UpdateNotificationResponse, error) {
	if wsc.updateNotificationResponseFlag {
		return updateNotificationAccepted()
	} else {
		return updateNotificationRejected()

	}
}

// set default response for UpdateNotification. True sets wallet to accep all channel updates. False rejects all channel updates
func (wsc *MyWalletServiceClient) SetUpdateNotificationResponse(flag bool) {
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

func (wsc *MyWalletServiceClient) SignMessage(ctx context.Context, in *proto.SignMessageRequest, opts ...grpc.CallOption) (*proto.SignMessageResponse, error) {
	if wsc.signMessageResponseFlag {
		return wsc.signMessageAccepted(in.Data)
	} else {
		return wsc.signMessageRejected()
	}
}

// set default response for SignMessage. True sets wallet to sign all messages, false rejects all requests to sign message
func (wsc *MyWalletServiceClient) SetSignMessageResponse(flag bool) {
	wsc.signMessageResponseFlag = flag
}

func (wsc *MyWalletServiceClient) signMessageAccepted(data []byte) (*proto.SignMessageResponse, error) {
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

func (wsc *MyWalletServiceClient) signMessageRejected() (*proto.SignMessageResponse, error) {
	return &proto.SignMessageResponse{
		Msg: &proto.SignMessageResponse_Rejected{
			Rejected: &proto.Rejected{
				Reason: "Not accepting messages",
			},
		}}, nil
}

func (wsc *MyWalletServiceClient) SignTransaction(ctx context.Context, tx *proto.SignTransactionRequest) (*proto.SignTransactionResponse, error) {
	if wsc.signTransactionResponseFlag {
		return wsc.signedTransactionAccepted(tx)
	} else {
		return wsc.signedTransationRejected()
	}
}

// set default response for SignTransaction. True sets wallet to sign all tx requests, and false to reject all tx requests
func (wsc *MyWalletServiceClient) SetSignTransactionResponse(flag bool) {
	wsc.signMessageResponseFlag = flag
}

func (wsc *MyWalletServiceClient) signedTransactionAccepted(tx *proto.SignTransactionRequest) (*proto.SignTransactionResponse, error) {
	ckbAddr := address.AsParticipant(wsc.account.Address()).ToCKBAddress(wsc.network)
	txSigner := backend.NewSignerInstance(ckbAddr, *wsc.privateKey, types.NetworkTest)
	txWithScriptGroups := &transaction.TransactionWithScriptGroups{}
	err := json.Unmarshal(tx.Transaction, txWithScriptGroups)
	if err != nil {
		log.Println("Error unmarshalling transaction", err)
	}
	signedTx, err := txSigner.SignTransaction(txWithScriptGroups)
	if err != nil {
		log.Println("Error signing transaction", err)
	}
	signedTxBytes, err := json.Marshal(signedTx)
	if err != nil {
		log.Println("Error marshalling signed transaction", err)
	}
	return &proto.SignTransactionResponse{
		Msg: &proto.SignTransactionResponse_Transaction{
			Transaction: signedTxBytes,
		},
	}, nil
}

func (wsc *MyWalletServiceClient) signedTransationRejected() (*proto.SignTransactionResponse, error) {
	return &proto.SignTransactionResponse{
		Msg: &proto.SignTransactionResponse_Rejected{
			Rejected: &proto.Rejected{
				Reason: "Not accepting transactions",
			},
		},
	}, nil

}

// sets default response for GetAssets().
func (wsc *MyWalletServiceClient) SetGetAssetsResponse(flag bool) {}
