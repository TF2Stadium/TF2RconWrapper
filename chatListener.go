package TF2RconWrapper

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

type rawMessage struct {
	data []byte
	n    int //length
}

// RconChatListener maintains an UDP server that receives redirected chat messages from TF2 servers
type RconChatListener struct {
	conn        *net.UDPConn
	servers     map[string]*ServerListener
	serversLock *sync.RWMutex
	addr        *net.UDPAddr
	localip     string
	port        string
	rng         *rand.Rand

	rawMessage chan rawMessage
}

// NewRconChatListener builds a new RconChatListener. Its arguments are localip (the ip of this server) and
// port (the port the listener will use)
func NewRconChatListener(localip, port string) (*RconChatListener, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}

	servers := make(map[string]*ServerListener)

	rng := rand.New(rand.NewSource(time.Now().Unix()))

	listener := &RconChatListener{nil, servers, new(sync.RWMutex), addr, localip, port, rng, make(chan rawMessage)}
	listener.startListening()
	return listener, nil
}

func (r *RconChatListener) startListening() {
	conn, err := net.ListenUDP("udp", r.addr)
	r.conn = conn
	if err != nil {
		log.Println(err)
		return
	}
	conn.SetReadBuffer(1048576)
	go r.processStrings()
	go r.readStrings()
}

func (r *RconChatListener) processStrings() {
	for {
		select {
		case raw := <-r.rawMessage:
			message := raw.data[0:raw.n]

			secret, err := getSecret(message)

			r.serversLock.RLock()
			s, ok := r.servers[secret]
			r.serversLock.RUnlock()
			if !ok {
				continue
			}

			messageObj, err := proccessMessage(message)

			if err != nil {
				log.Println(err)
				continue
			}

			s.Messages <- messageObj
		}
	}
}

func (r *RconChatListener) readStrings() {
	buff := make([]byte, 65000)

	for {
		n, _, err := r.conn.ReadFromUDP(buff)
		if err != nil {
			fmt.Println("Error receiving server chat data: ", err)
			continue
		}

		r.rawMessage <- rawMessage{buff, n}
	}
}

// Close stops the RconChatListener
func (r *RconChatListener) Close(m *TF2RconConnection) {
	m.StopLogRedirection(r.localip, r.port)

	r.serversLock.Lock()
	s := r.servers[m.secret]
	delete(r.servers, m.secret)
	r.serversLock.Unlock()

	<-s.Messages
	close(s.Messages)
}

// CreateServerListener creates a ServerListener that receives chat messages from a
// particular TF2 server
func (r *RconChatListener) CreateServerListener(m *TF2RconConnection) *ServerListener {

	secret := strconv.Itoa(r.rng.Intn(999998) + 1)

	r.serversLock.RLock()
	_, ok := r.servers[secret]
	for ok {
		secret = strconv.Itoa(r.rng.Intn(999998) + 1)
		_, ok = r.servers[secret]
	}
	m.secret = secret
	r.serversLock.RUnlock()

	s := &ServerListener{make(chan LogMessage), m.host, secret, r}

	r.serversLock.Lock()
	r.servers[secret] = s
	r.serversLock.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(r.localip, r.port)
	return s
}

// ServerListener represents a listener that receives chat messages from a particular
// TF2 server. It's built and managed by an RconChatListener instance.
type ServerListener struct {
	Messages chan LogMessage
	host     string
	secret   string
	listener *RconChatListener
}
