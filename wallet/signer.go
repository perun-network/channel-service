package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/v2/address"
	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
)

type RemoteSigner struct {
	wcs  proto.WalletServiceClient
	addr address.Address
}

func NewRemoteSigner(wcs proto.WalletServiceClient, addr address.Address) *RemoteSigner {
	return &RemoteSigner{
		wcs:  wcs,
		addr: addr,
	}
}

func (s RemoteSigner) SignTransaction(tx *transaction.TransactionWithScriptGroups) (*types.Transaction, error) {
	scriptBytes, err := json.Marshal(s.addr.Script)
	if err != nil {
		return nil, err
	}
	txBytes, err := json.Marshal(tx.TxView)
	if err != nil {
		return nil, err
	}
	log.Printf("Signing transaction: %x\n", txBytes)
	req := &proto.SignTransactionRequest{
		Identifier:  scriptBytes, // TODO: Maybe encode network also?
		Transaction: txBytes,
	}
	resp, err := s.wcs.SignTransaction(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	if rej := resp.GetRejected(); rej != nil {
		return nil, fmt.Errorf("transaction signing failed: %s", rej.Reason)
	}

	var signedTx types.Transaction
	signedTxBytes := resp.GetTransaction()
	fmt.Printf("Signed transaction: %s\n", string(signedTxBytes))
	if err = json.Unmarshal(signedTxBytes, &signedTx); err != nil {
		return nil, err
	}
	return &signedTx, nil
}

func (s RemoteSigner) Address() address.Address {
	return s.addr
}
