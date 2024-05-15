package test

import (
	"encoding/hex"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"
	"perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/backend"
)

func setup() {

	/*
		alicePrivateKey,err := secp256k1.GeneratePrivateKey()
		if err != nil {
			fmt.Println("Error generating Alice's private key")
		}
		alicePubKey := alicePrivateKey.PubKey()

		bobPrivateKey,err := secp256k1.GeneratePrivateKey()
		if err != nil {
			fmt.Println("Error generating Bob's private key")
		}
		bobPubKey := bobPrivateKey.PubKey()
		pubKeys := []*secp256k1.PublicKey{alicePubKey, bobPubKey}
	*/
	alicePrivateKey := "6a5d968cbd4afbbeb57506252228c9c393b2fff5c862f8c645c48ad7014829d6"
	bobPrivateKey := "73cbbe60963a42db5752c1f0e112cdac2a60ac3624b0d9baa61cc1cb3aee34cd"

	getKeys := func() ([]secp256k1.PublicKey, error) {
		alicePrivateKeyBytes, err := hex.DecodeString(alicePrivateKey)
		if err != nil {
			return nil, err
		}
		alicePrivateKey := secp256k1.PrivKeyFromBytes(alicePrivateKeyBytes)

		alicePubKey := alicePrivateKey.PubKey()

		bobPrivateKeyBytes, err := hex.DecodeString(bobPrivateKey)
		if err != nil {
			return nil, err
		}
		bobPrivateKey := secp256k1.PrivKeyFromBytes(bobPrivateKeyBytes)

		bobPubKey := bobPrivateKey.PubKey()

		pubKeys := []secp256k1.PublicKey{*alicePubKey, *bobPubKey}
		return pubKeys, nil
	}
	pubKeys, err := getKeys()
	if err != nil {
		fmt.Println("Error creating public keys")
	}

	parts, err := MakeParticipants(pubKeys)
	if err != nil {
		fmt.Println("Error making participants")
	}

	wAliceAddr, err := MakeDefaultWireAddress(parts[0])
	if err != nil {
		fmt.Println("Error while creating default address for Alice")
	}

	wBobAddr, err := MakeDefaultWireAddress(parts[1])
	if err != nil {
		fmt.Println("Error while creating default address for Bob")
	}

	bus := wire.NewLocalBus()
	// setup up funder and adjudicator
	// to do this I need a remoteSigner
	signer := backend.NewSignerInstance()
}

func MakeParticipants(pks []secp256k1.PublicKey) ([]address.Participant, error) {
	parts := make([]address.Participant, len(pks))
	for i := range pks {
		part, err := address.NewDefaultParticipant(&pks[i])
		if err != nil {
			return nil, fmt.Errorf("unable to create participant: %w", err)
		}
		parts[i] = *part
	}
	return parts, nil
}

func MakeDefaultWireAddress(participant address.Participant) (wire.Address, error) {
	ckbAddr, err := participant.ToCKBAddress(types.NetworkTest).Encode()
	if err != nil {
		return nil, err
	}
	return simple.NewAddress(ckbAddr), nil
}
