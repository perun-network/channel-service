package wallet

import (
	"context"
	"perun.network/channel-service/rpc/proto"
	"perun.network/perun-ckb-backend/wallet/address"
)

type ExternalClient struct {
	c proto.WalletServiceClient
}

func (e ExternalClient) Unlock(participant address.Participant) error {
	return nil
}

func (e ExternalClient) SignData(participant address.Participant, data []byte) ([]byte, error) {
	sm := &proto.SignMessageRequest{Pubkey: participant.PubKey.SerializeCompressed(), Data: data}
	smr, err := e.c.SignMessage(context.TODO(), sm)
	if err != nil {
		return nil, err
	}
	// We assume that the wallet returns a PaddedSignature (see perun-ckb-backend/wallet/signature.go).
	return smr.GetSignature(), nil
}

func NewExternalClient(c proto.WalletServiceClient) *ExternalClient {
	return &ExternalClient{c: c}
}
