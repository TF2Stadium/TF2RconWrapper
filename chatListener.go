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

type RawMessage struct {
	data []byte
	n    int //length
}

func (r RawMessage) String() string {
	return string(r.data)
}

// RconChatListener maintains an UDP server that receives redirected chat messages from TF2 servers
type RconChatListener struct {
	conn        *net.UDPConn
	servers     map[string]*ServerListener
	serversLock *sync.RWMutex
	wait        *sync.WaitGroup
	addr        *net.UDPAddr
	localip     string
	port        string
	rng         *rand.Rand
}

// NewRconChatListener builds a new RconChatListener. Its arguments are localip (the ip of this server) and
// port (the port the listener will use)
func NewRconChatListener(localip, port string) (*RconChatListener, error) {
	addr, err := net.ResolveUDPAddr("udp4", ":"+port)
	if err != nil {
		return nil, err
	}

	servers := make(map[string]*ServerListener)

	rng := rand.New(rand.NewSource(time.Now().Unix()))

	listener := &RconChatListener{nil, servers, new(sync.RWMutex), new(sync.WaitGroup), addr, localip, port, rng}
	listener.startListening()
	return listener, nil
}

func (r *RconChatListener) startListening() {
	conn, err := net.ListenUDP("udp", r.addr)
	r.conn = conn
	if err != nil {
		log.Fatal(err)
		return
	}
	conn.SetReadBuffer(1048576)
	go r.readStrings()
}

func (r *RconChatListener) readStrings() {
	rawMessageC := make(chan RawMessage, 500)

	go func() {
		for {
			raw := <-rawMessageC
			r.wait.Wait()
			message := raw.data[0:raw.n]
			secret, err := getSecret(message)
			if err != nil {
				continue
			}

			r.serversLock.RLock()
			s, ok := r.servers[secret]
			r.serversLock.RUnlock()
			if ok {
				s.RawMessages <- raw
			}
		}
	}()

	for {
		buff := make([]byte, 65000)
		n, err := r.conn.Read(buff)
		//log.Println(string(buff[0:n]))

		if err != nil {
			fmt.Println("Error receiving server chat data: ", err)
			continue
		}

		rawMessageC <- RawMessage{buff, n}
	}
}

// Close stops the RconChatListener
func (s *ServerListener) Close(m *TF2RconConnection) {
	s.listener.wait.Add(1)
	delete(s.listener.servers, s.Secret)
	s.listener.wait.Done()

	m.StopLogRedirection(s.listener.localip, s.listener.port)
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
	r.serversLock.RUnlock()

	return r.CreateListenerWithSecret(m, secret)
}

func (r *RconChatListener) CreateListenerWithSecret(m *TF2RconConnection, secret string) *ServerListener {
	s := &ServerListener{make(chan RawMessage, 10), m.host, secret, r}

	r.serversLock.Lock()
	r.servers[secret] = s
	//log.Println(r.servers)
	r.serversLock.Unlock()

	m.Query("sv_logsecret " + secret)
	m.RedirectLogs(r.localip, r.port)
	return s
}

// ServerListener represents a listener that receives chat messages from a particular
// TF2 server. It's built and managed by an RconChatListener instance.
type ServerListener struct {
	RawMessages chan RawMessage

	host     string
	Secret   string
	listener *RconChatListener
}
