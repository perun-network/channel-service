package service

import (
	"errors"
	"sync"

	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
)

var (
	// ErrAddrExists is returned when an address already exists in the resolver.
	ErrAddrExists = errors.New("address already exists")
	// ErrAddrNotFound is returned when an address is not found in the resolver.
	ErrAddrNotFound = errors.New("address not found")
)

// AddressResolver resolves wallet addresses to wire addresses.
type AddressResolver interface {
	// GetWireAddress returns the wire address for the given wallet address.
	GetWireAddress(walletAddr wallet.Address) (wire.Address, error)

	// AddWire adds the wire address to the register and returns it. If the wire address already exists, it returns the existing wire address.
	AddWire(walletAddr wallet.Address, wireAddr wire.Address) (wire.Address, error)

	// SetWire sets the wire address for the given wallet address.
	SetWire(walletAddr wallet.Address, wireAddr wire.Address) error

	// DeleteWire deletes the wire address from the register.
	DeleteWire(walletAddr wallet.Address) error
}

// MutexLocalAddressResolver is a thread-safe implementation of the AddressResolver interface.
type MutexLocalAddressResolver struct {
	mtx      sync.Mutex
	register map[wallet.AddrKey]wire.Address
}

// GetWireAddress returns the wire address for the given wallet address.
func (m *MutexLocalAddressResolver) GetWireAddress(walletAddr wallet.Address) (wire.Address, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.register[wallet.Key(walletAddr)]
	if !ok {
		return nil, ErrAddrNotFound
	}
	return u, nil
}

// AddWire adds the wire address to the register and returns it. If the wire address already exists, it returns the existing wire address.
func (m *MutexLocalAddressResolver) AddWire(walletAddr wallet.Address, wireAddr wire.Address) (wire.Address, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	/*
		if wire, ok := m.register[wallet.Key(walletAddr)]; ok {
			return wire, ErrAddrExists
		}
	*/
	addrKey := wallet.Key(walletAddr)
	if wireV, ok := m.register[addrKey]; ok { // V in wireV means nothing. It is just there because we already have a pacakge called wire
		return wireV, ErrAddrExists
	}
	m.register[wallet.Key(walletAddr)] = wireAddr
	return wireAddr, nil
}

// SetWire sets the wire address for the given wallet address.
func (m *MutexLocalAddressResolver) SetWire(walletAddr wallet.Address, wireAddr wire.Address) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.register[wallet.Key(walletAddr)] = wireAddr
	return nil
}

// DeleteWire deletes the wire address from the register.
func (m *MutexLocalAddressResolver) DeleteWire(walletAddr wallet.Address) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.register, wallet.Key(walletAddr))
	return nil
}

// NewMutexLocalAddressResolver creates a new MutexLocalAddressResolver.
func NewMutexLocalAddressResolver() *MutexLocalAddressResolver {
	return &MutexLocalAddressResolver{
		register: make(map[wallet.AddrKey]wire.Address),
	}
}
