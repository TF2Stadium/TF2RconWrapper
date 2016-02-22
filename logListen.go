package TF2RconWrapper

import (
	"bytes"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

type EventListener struct {
	PlayerConnected      func(PlayerData)
	PlayerDisconnected   func(PlayerData)
	PlayerGlobalMessage  func(PlayerData, string) // strings are chat message
	PlayerTeamMessage    func(PlayerData, string)
	PlayerSpawned        func(PlayerData, string) // string is class
	PlayerClassChanged   func(PlayerData, string) // string is new classes
	PlayerTeamChange     func(PlayerData, string) // string is new team
	PlayerKilled         func(PlayerKill)
	PlayerDamaged        func(PlayerDamage)
	PlayerHealed         func(PlayerHeal)
	PlayerKilledMedic    func(PlayerTrigger)
	PlayerUberFinished   func(PlayerData)
	PlayerBlockedCapture func(CPData, PlayerData) // cp blocked by player
	PlayerItemPickup     func(ItemPickup)
	TeamPointCapture     func(TeamData)
	TeamScoreUpdate      func(TeamData)
	GameOver             func()
	WorldRoundWin        func(string) // string is team which won
	CVarChange           func(variable string, value string)
	LogFileClosed        func()
}

type Listener struct {
	mapMu    *sync.RWMutex
	sources  map[string]*Source
	channels map[string](chan string)

	listenAddr   *net.UDPAddr
	redirectAddr string
}

type Source struct {
	Secret string
	logsMu *sync.RWMutex //protects logs
	logs   *bytes.Buffer

	handlerMu *sync.Mutex //protects handler and closed
	handler   *EventListener
	closed    bool
}

func (s *Source) Logs() *bytes.Buffer {
	return s.logs
}

// NewListener returns a new Listener
func NewListener(addr string) (*Listener, error) {
	return NewListenerAddr(addr, addr)
}

func NewListenerAddr(port, redirectAddr string) (*Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		mapMu:    new(sync.RWMutex),
		sources:  make(map[string]*Source),
		channels: make(map[string](chan string)),

		listenAddr:   addr,
		redirectAddr: redirectAddr,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	go l.start(conn)
	return l, nil
}

func (l *Listener) RemoveSource(s *Source, m *TF2RconConnection) {
	s.handlerMu.Lock()
	s.closed = true
	s.handlerMu.Unlock()

	l.mapMu.Lock()
	delete(l.sources, s.Secret)
	l.mapMu.Unlock()

	m.StopLogRedirection(l.redirectAddr)
}

func (l *Listener) start(conn *net.UDPConn) {
	for {
		buff := make([]byte, 512)
		n, err := conn.Read(buff)
		if err != nil {
			log.Println(err)
		}

		secret, pos, err := getSecret(buff[0:n])
		if err != nil {
			continue
		}

		l.mapMu.RLock()
		source, ok := l.sources[secret]
		l.mapMu.RUnlock()

		if !ok {
			continue
		}

		go func() {
			source.handlerMu.Lock()
			defer source.handlerMu.Unlock()

			if source.closed {
				return
			}

			source.logsMu.Lock()
			source.logs.Write(buff[11 : n-1])
			source.logsMu.Unlock()

			handler := source.handler

			m := parse(string(buff[pos : n-2]))

			m.Parsed.CallHandler(handler)
		}()
	}
}

func (l *Listener) AddSource(handler *EventListener, m *TF2RconConnection) *Source {
	secret := strconv.Itoa(100000 + rand.Intn(800000))
	rand.Seed(time.Now().Unix())

	l.mapMu.RLock()
	_, ok := l.sources[secret]
	for ok {
		secret = strconv.Itoa(100000 + rand.Intn(800000))
		_, ok = l.sources[secret]
	}
	l.mapMu.RUnlock()

	return l.AddSourceSecret(secret, handler, m)
}

func (l *Listener) AddSourceSecret(secret string, handler *EventListener, m *TF2RconConnection) *Source {
	s := newSource(secret, handler)

	l.mapMu.Lock()
	l.sources[secret] = s
	l.mapMu.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(l.redirectAddr)
	return s
}

func newSource(secret string, handler *EventListener) *Source {
	return &Source{
		Secret: secret,
		logsMu: new(sync.RWMutex),
		logs:   new(bytes.Buffer),

		handlerMu: new(sync.Mutex),
		handler:   handler,
	}
}
