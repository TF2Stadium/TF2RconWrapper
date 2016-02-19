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

type Handler interface {
	PlayerConnected(PlayerData)
	PlayerDisconnected(PlayerData)

	PlayerGlobalMessage(PlayerData, string) // strings are chat message
	PlayerTeamMessage(PlayerData, string)

	PlayerSpawned(PlayerData, string)     // string is class
	PlayerClassChange(PlayerData, string) // string is new classes
	PlayerTeamChange(PlayerData, string)  // string is new team

	PlayerKilled(PlayerKill)
	PlayerDamaged(PlayerDamage)
	PlayerHealed(PlayerHeal)
	PlayerKilledMedic(PlayerTrigger)
	PlayerUberFinished(PlayerData)
	PlayerBlockedCapture(PlayerData)
	PlayerItemPickup(ItemPickup)

	TeamPointCapture(TeamData)
	TeamScoreUpdate(TeamData)
	GameOver()

	WorldRoundWin(string) // string is team which won

	CVarChange(variable string, value string)
	LogFileClosed()
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
	handler   Handler
	closed    bool
}

func (s *Source) Logs() *bytes.Buffer {
	s.logsMu.RLock()
	b := s.logs.Bytes()
	var logs []byte
	copy(b, logs)
	s.logsMu.RUnlock()

	return bytes.NewBuffer(logs)
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
			source.logs.WriteString("L ")
			source.logs.Write(buff[pos : n-2])
			source.logs.WriteByte('\n')
			source.logsMu.Unlock()

			handler := source.handler

			m := parse(string(buff[pos : n-2]))

			m.Parsed.CallHandler(handler)
		}()
	}
}

func (l *Listener) AddSource(handler Handler, m *TF2RconConnection) *Source {
	secret := strconv.Itoa(rand.Intn(999998) + 1)
	rand.Seed(time.Now().Unix())

	l.mapMu.RLock()
	_, ok := l.sources[secret]
	for ok {
		secret = strconv.Itoa(rand.Intn(999998) + 1)
		_, ok = l.sources[secret]
	}
	l.mapMu.RUnlock()

	return l.AddSourceSecret(secret, handler, m)
}

func (l *Listener) AddSourceSecret(secret string, handler Handler, m *TF2RconConnection) *Source {
	s := newSource(secret, handler)

	l.mapMu.Lock()
	l.sources[secret] = s
	l.mapMu.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(l.redirectAddr)
	return s
}

func newSource(secret string, handler Handler) *Source {
	return &Source{
		Secret: secret,
		logsMu: new(sync.RWMutex),
		logs:   new(bytes.Buffer),

		handlerMu: new(sync.Mutex),
		handler:   handler,
	}
}
