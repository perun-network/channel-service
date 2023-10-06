package wallet

import (
	"context"
	"perun.network/channel-service/rpc"
	"perun.network/perun-ckb-backend/wallet/address"
)

type ExternalClient struct {
	c rpc.WalletServiceClient
}

func (e ExternalClient) Unlock(participant address.Participant) error {
	return nil
}

func (e ExternalClient) SignData(participant address.Participant, data []byte) ([]byte, error) {
	sm := &rpc.SignMessageRequest{Data: data}
	// TODO: SignMessage needs some reference to the participant / key to sign with.
	smr, err := e.c.SignMessage(context.TODO(), sm)
	if err != nil {
		return nil, err
	}
	return smr.GetSignature(), nil
}

func NewExternalClient(c rpc.WalletServiceClient) *ExternalClient {
	return &ExternalClient{c: c}
}
