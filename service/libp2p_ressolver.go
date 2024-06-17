package service

import (
	"errors"

	"github.com/perun-network/perun-libp2p-wire/p2p"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
)

var (
	// ErrAddrNotMatch is returned when the wire address does not match the account address.
	ErrAddrNotMatch = errors.New("wire address does not match account address")
)

// RelayServerResolver resolves the wallet address to a wire address using Relay Server.
type RelayServerResolver struct {
	*p2p.Account
}

// NewRelayServerResolver creates a new RelayServerResolver.
func NewRelayServerResolver(acc *p2p.Account) *RelayServerResolver {
	return &RelayServerResolver{acc}
}

// GetWireAddress returns the wire address for the given wallet address.
func (r *RelayServerResolver) GetWireAddress(walletAddr wallet.Address) (wire.Address, error) {
	waddr, err := r.QueryOnChainAddress(walletAddr)
	if err != nil {
		return nil, ErrAddrNotFound
	}

	return waddr, nil
}

// AddWire adds the wire address to the register and returns it. If the wire address already exists, it returns the existing wire address.
func (r *RelayServerResolver) AddWire(walletAddr wallet.Address, wireAddr wire.Address) (wire.Address, error) {
	if !r.Address().Equal(wireAddr) {
		return nil, ErrAddrNotMatch
	}

	if _, err := r.QueryOnChainAddress(walletAddr); err == nil {
		return wireAddr, ErrAddrExists
	}

	err := r.RegisterOnChainAddress(walletAddr)
	if err != nil {
		return nil, err

	}

	return wireAddr, nil
}

// SetWire sets the wire address for the given wallet address.
func (r *RelayServerResolver) SetWire(walletAddr wallet.Address, wireAddr wire.Address) error {
	if !r.Address().Equal(wireAddr) {
		return ErrAddrNotMatch
	}

	err := r.RegisterOnChainAddress(walletAddr)
	if err != nil {
		return err

	}

	return nil
}

// DeleteWire deletes the wire address from the register.
func (r *RelayServerResolver) DeleteWire(walletAddr wallet.Address) error {
	return r.DeregisterOnChainAddress(walletAddr)
}
