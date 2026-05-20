package handler

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type FriendStatus int

const (
	Alive FriendStatus = iota
	Dead
)

type Friend struct {
	mu      sync.Mutex
	ID      string
	Port    int
	Address string
	Name    string
	Status  FriendStatus
	lastAck time.Time
}
type Member struct {
	Friends map[string]*Friend
	mu      sync.Mutex
	ID      string
	Port    int
	Address string
	Name    string
	log     log.Logger
}

func NewMember(
	port int,
	address string,
	name string,
) *Member {
	now := time.Now()
	id := now.UnixMilli()
	logger := log.New(
		os.Stdout,
		"[Member] ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	return &Member{
		ID:      strconv.FormatInt(id, 10),
		Port:    port,
		Address: address,
		Name:    name,
		Friends: map[string]*Friend{},
		log:     *logger,
	}
}

func (m *Member) AddFriend(id, Address, name string, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	friend := Friend{
		ID:      id,
		Address: Address,
		Name:    name,
		Port:    port,
		lastAck: time.Now(),
	}
	m.Friends[id] = &friend
}

func (m *Member) UpdateFriendStatus(id string, stat FriendStatus) {
	switch stat {
	case Alive:
		m.alive(id)
	case Dead:
		m.killFriend(id)
	}

}

func (m *Member) killFriend(id string) {
	friend, ok := m.Friends[id]
	if !ok {
		m.log.Printf("No Friend with this id %s", id)
		return
	}
	friend.mu.Lock()
	defer friend.mu.Unlock()
	friend.Status = Dead

}
func (m *Member) alive(id string) {
	friend, ok := m.Friends[id]
	if !ok {
		m.log.Printf("No Friend with this id %s", id)
		return
	}
	friend.mu.Lock()
	defer friend.mu.Unlock()
	friend.lastAck = time.Now()
	friend.Status = Alive
}
