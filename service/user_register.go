package service

import (
	"errors"
	"log"
	"net"
	"sync"

	"perun.network/go-perun/channel"
	"perun.network/perun-ckb-backend/wallet/address"
)

var ErrUserNotFound = errors.New("user not found")

// UserRegister connects channel.IDs to Users.
type UserRegister interface {
	// GetUser returns the user associated with the channel or an ErrUserNotFound error, if no such user is registered.
	GetUser(channel.ID) (*User, error)
	AssignChannelID(channel.ID, *User) error
	// GetUserFromParticipant returns the user associated with the participant or an ErrUserNotFound error, if no such
	// user is registered.
	GetUserFromParticipant(participant address.Participant) (*User, error)
	// AddUser adds the user to the register and returns it. If the user already exists, it returns the existing user.
	AddUser(participant address.Participant, user *User) *User
}

type MutexUserRegister struct {
	mtx      sync.Mutex
	register map[channel.ID]*User
	// FIXME: Use a clone or key method on address.Participant instead of only
	// pubkey.
	users map[[33]byte]*User
}

func (m *MutexUserRegister) GetUserFromParticipant(participant address.Participant) (*User, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.users[participant.GetCompressedSEC1()]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *MutexUserRegister) AddUser(participant address.Participant, user *User) *User {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.users[participant.GetCompressedSEC1()]
	if ok {
		return u
	}
	log.Printf("Adding user %v to register", participant)
	m.users[participant.GetCompressedSEC1()] = user
	return user
}

func (m *MutexUserRegister) GetUser(cid channel.ID) (*User, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.register[cid]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *MutexUserRegister) AssignChannelID(cid channel.ID, user *User) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	_, ok := m.register[cid]
	if ok {
		return errors.New("channel already exists")
	}
	m.register[cid] = user
	return nil
}

func NewMutexUserRegister() *MutexUserRegister {
	return &MutexUserRegister{
		register: make(map[channel.ID]*User),
		users:    make(map[[33]byte]*User),
	}
}

// NetworkRegister is a hack to connect users with their network address to make a single channels service work with
// multiple neuron wallets for demo purposes.
type NetworkRegister interface {
	AddUser(addr net.Addr, user *User) *User
	GetUser(addr net.Addr) (*User, error)
}

type MutexNetworkRegister struct {
	mtx      sync.Mutex
	register map[string]*User
}

func NewMutexNetworkRegister() *MutexNetworkRegister {
	return &MutexNetworkRegister{
		mtx:      sync.Mutex{},
		register: make(map[string]*User),
	}
}

func (m *MutexNetworkRegister) AddUser(addr net.Addr, user *User) *User {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.register[addr.String()]
	if ok {
		return u
	}
	m.register[addr.String()] = user
	return u
}

func (m *MutexNetworkRegister) GetUser(addr net.Addr) (*User, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	u, ok := m.register[addr.String()]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}
