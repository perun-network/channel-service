package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	addr2 "github.com/nervosnetwork/ckb-sdk-go/v2/address"
	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/rpc/proto"
	"perun.network/perun-ckb-backend/wallet/address"
)

type RemoteSigner struct {
	wcs         proto.WalletServiceClient
	addr        addr2.Address
	participant *address.Participant
}

func NewRemoteSigner(wcs proto.WalletServiceClient, addr addr2.Address, part *address.Participant) *RemoteSigner {
	return &RemoteSigner{
		wcs:         wcs,
		addr:        addr,
		participant: part,
	}
}

func (s *RemoteSigner) PublicKey() *secp256k1.PublicKey {
	if s.participant == nil {
		log.Panic("RemoteSigner participant is nil")
	}
	return s.participant.PubKey
}

func (s RemoteSigner) SignTransaction(tx *transaction.TransactionWithScriptGroups) (*types.Transaction, error) {
	scriptBytes, err := json.Marshal(s.addr.Script)
	if err != nil {
		return nil, err
	}

	txBytes, err := json.Marshal(tx)

	if err != nil {
		return nil, err
	}
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
	if err = json.Unmarshal(signedTxBytes, &signedTx); err != nil {
		return nil, err
	}
	return &signedTx, nil
}

func (s RemoteSigner) Address() addr2.Address {
	return s.addr
}
