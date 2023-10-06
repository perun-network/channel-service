package service

import (
	"errors"
	"perun.network/go-perun/channel"
	"sync"
)

var ErrUserNotFound = errors.New("user not found")

// UserRegister connects channel.IDs to Users.
type UserRegister interface {
	GetUser(channel.ID) (*User, error)
	AssignChannelID(channel.ID, *User) error
}

type MutexUserRegister struct {
	mtx      sync.Mutex
	register map[channel.ID]*User
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
	}
}
