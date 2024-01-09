package wallet

import (
	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
)

// RemoteTransaction is a transaction that is sent to a remote signer and
// handles the conversion between said remote signer and the local domain.
type RemoteTransaction struct{}

// NewRemoteTransaction creates a new remote transaction from
// `transaction.TransactionWithScriptGroups`.
func NewRemoteTransaction(tx *transaction.TransactionWithScriptGroups) *RemoteTransaction {
	return &RemoteTransaction{}
}

// Encodes the remote transaction in a way that can be understood by the
// WalletBackend service.
func (rt *RemoteTransaction) Encode() ([]byte, error) {
	// TODO: Encode and send as `types.Transaction`. This should be signable
	// by the remote signer.
	return nil, nil
}

// Decodes the remote transaction from a byte slice. Typically the response
// from the WalletBackend service.
func (rt *RemoteTransaction) Decode([]byte) error {
	return nil
}

// Raw tries to return the raw transaction, which might or might not be
// signed.
func (rt *RemoteTransaction) Raw() *types.Transaction {
	return nil
}
