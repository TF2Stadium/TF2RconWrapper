package TF2RconWrapper

import (
	"bytes"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
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
	TournamentStarted    func()
	RconCommand          func(from, command string) // from - IP Address, command - command executed

	success chan struct{}
}

type Listener struct {
	mapMu    *sync.RWMutex
	sources  map[string]*Source
	channels map[string](chan string)

	listenAddr   *net.UDPAddr
	redirectAddr string
	print        bool
}

type Source struct {
	Secret string
	logsMu *sync.RWMutex //protects logs
	logs   *bytes.Buffer

	handler *EventListener
	closed  *int32

	//fields used for test only
	test bool
	rcon *TF2RconConnection
}

func (s *Source) Logs() *bytes.Buffer {
	s.logsMu.RLock()
	slice := make([]byte, s.logs.Len())
	s.logs.Read(slice)
	logs := bytes.NewBuffer(slice)
	s.logsMu.RUnlock()

	return logs
}

// NewListener returns a new Listener
func NewListener(addr string, print bool) (*Listener, error) {
	return NewListenerAddr(addr, addr, print)
}

func NewListenerAddr(port, redirectAddr string, print bool) (*Listener, error) {
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
		print:        print,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	go l.start(conn)
	return l, nil
}

func (l *Listener) RemoveSource(s *Source, m *TF2RconConnection) {
	atomic.StoreInt32(s.closed, 1)

	l.mapMu.Lock()
	delete(l.sources, s.Secret)
	l.mapMu.Unlock()

	m.StopLogRedirection(l.redirectAddr)
}

func (l *Listener) start(conn *net.UDPConn) {
	for {
		buff := make([]byte, 2048)
		n, err := conn.Read(buff)
		if err != nil {
			log.Println(err)
		}

		secret, Lpos, err := getSecret(buff[0:n])
		if err != nil {
			continue
		}

		if l.print {
			log.Println(string(buff[0 : n-1]))
		}

		l.mapMu.RLock()
		source, ok := l.sources[secret]
		l.mapMu.RUnlock()

		if !ok {
			continue
		}

		go func() {
			if atomic.LoadInt32(source.closed) == 1 {
				return
			}

			handler := source.handler

			if source.test {
				l.mapMu.Lock()
				delete(l.sources, source.Secret)
				l.mapMu.Unlock()

				atomic.StoreInt32(source.closed, 1)

				handler.success <- struct{}{}

				source.rcon.StopLogRedirection(l.redirectAddr)
				source.rcon.Close()
				return
			}

			source.logsMu.Lock()
			source.logs.Write(buff[Lpos : n-1])
			source.logsMu.Unlock()

			m := ParseLogEntry(string(buff[Lpos : n-2]))
			m.Parsed.CallHandler(handler)
		}()
	}
}

func (l *Listener) getSecret() string {
	secret := strconv.FormatUint(uint64(rand.Int63()+1), 10)
	rand.Seed(time.Now().Unix())

	l.mapMu.RLock()
	_, ok := l.sources[secret]
	for ok {
		secret = strconv.Itoa(100000 + rand.Intn(800000))
		_, ok = l.sources[secret]
	}
	l.mapMu.RUnlock()

	return secret
}

func (l *Listener) TestSource(m *TF2RconConnection) bool {
	secret := l.getSecret()
	e := &EventListener{success: make(chan struct{}, 1)}

	s := newSource(secret, e, true)
	s.rcon = m
	l.mapMu.Lock()
	l.sources[secret] = s
	l.mapMu.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(l.redirectAddr)

	tick := time.After(5 * time.Second)
	select {
	case <-tick:
		return false
	case <-e.success:
		return true
	}
}

func (l *Listener) AddSource(handler *EventListener, m *TF2RconConnection) *Source {
	secret := l.getSecret()
	return l.AddSourceSecret(secret, handler, m)
}

func (l *Listener) AddSourceSecret(secret string, handler *EventListener, m *TF2RconConnection) *Source {
	s := newSource(secret, handler, false)

	l.mapMu.Lock()
	l.sources[secret] = s
	l.mapMu.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(l.redirectAddr)
	return s
}

func newSource(secret string, handler *EventListener, test bool) *Source {
	return &Source{
		Secret:  secret,
		logsMu:  new(sync.RWMutex),
		logs:    new(bytes.Buffer),
		handler: handler,
		closed:  new(int32),
		test:    test,
	}
}
