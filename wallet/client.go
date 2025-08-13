package wallet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
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
	// TODO: Inject types.NetworkType.
	addr, err := participant.ToCKBAddress(types.NetworkTest).Encode()
	if err != nil {
		panic(fmt.Sprintf("encoding participant addr: %v", err))
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("length of data to be signed is less than 4 bytes: %d", len(data))
	}
	tempIDLength := binary.BigEndian.Uint32(data[:4])
	if tempIDLength != 32 {
		return nil, errors.New("invalid TempChannelID length, must be 32 bytes")
	}
	tempIDBinary := data[4 : 4+tempIDLength]
	if err != nil {
		log.Println("Error converting data to TempChannelID", err)
		return nil, err
	}
	data1 := data[4+tempIDLength:]
	sm := &proto.SignMessageRequest{Pubkey: []byte(addr), Data: data1, TempChannelID: tempIDBinary}
	smr, err := e.c.SignMessage(context.TODO(), sm)
	if err != nil {
		return nil, err
	}
	if rejErr := smr.GetRejected(); rejErr != nil {
		return nil, fmt.Errorf("signing data: %s", rejErr.Reason)
	}
	// We assume that the wallet returns a PaddedSignature (see perun-ckb-backend/wallet/signature.go).
	return smr.GetSignature(), nil
}

func NewExternalClient(c proto.WalletServiceClient) *ExternalClient {
	return &ExternalClient{c: c}
}
