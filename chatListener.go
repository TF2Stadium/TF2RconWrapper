package TF2RconWrapper

import (
	"fmt"
	"log"
	"net"
	"time"
)

// ChatMessage represents a chat message in a TF2 server, and contains a timestamp and a message.
// The message can be a player message that contains the sender's username, steamid and other info
// or a server message.
type ChatMessage struct {
	Timestamp time.Time
	Message   string
}

func proccessMessage(text string) ChatMessage {
	text = text[7:]
	timeText := text[:21]
	message := text[23:]

	const refTime = "01/02/2006 -  15:04:05"

	timeObj, _ := time.Parse(refTime, timeText)

	return ChatMessage{timeObj, message}
}

// RconChatListener maintains an UDP server that receives redirected chat messages from TF2 servers
type RconChatListener struct {
	conn    *net.UDPConn
	servers map[string]*ServerListener
	exit    chan bool
	addr    *net.UDPAddr
	localip string
	port    string
}

// NewRconChatListener builds a new RconChatListener. Its arguments are localip (the ip of this server) and
// port (the port the listener will use)
func NewRconChatListener(localip, port string) (*RconChatListener, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}

	exit := make(chan bool)
	servers := make(map[string]*ServerListener)

	listener := &RconChatListener{nil, servers, exit, addr, localip, port}
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

	go r.readStrings()
}

func (r *RconChatListener) readStrings() {
	buff := make([]byte, 4096)

	for {
		select {
		case <-r.exit:
			return
		default:
			r.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, addr, err := r.conn.ReadFromUDP(buff)
			if err != nil {
				if typedErr, ok := err.(*net.OpError); ok && typedErr.Timeout() {
					continue
				}

				fmt.Println("Error receiving server chat data: ", err)
			}

			key := addr.String()

			s, ok := r.servers[key]

			if !ok {
				log.Println("Received chat info from an unregistered TF2 server")
				continue
			}

			message := string(buff[0:n])
			s.messages <- proccessMessage(message)
		}
	}
}

// Close stops the RconChatListener
func (r *RconChatListener) Close() {
	r.exit <- true
	r.conn.Close()
}

// CreateServerListener creates a ServerListener that receives chat messages from a
// particular TF2 server
func (r *RconChatListener) CreateServerListener(m *TF2RconConnection) *ServerListener {

	s := &ServerListener{make(chan ChatMessage), m.host, r}

	r.servers[m.host] = s

	go m.RedirectLogs(r.localip, r.port)

	return s
}

// ServerListener represents a listener that receives chat messages from a particular
// TF2 server. It's built and managed by an RconChatListener instance.
type ServerListener struct {
	messages chan ChatMessage
	host     string
	listener *RconChatListener
}

// GetNext blocks until a chat message is received and then returns it
func (s *ServerListener) GetNext() ChatMessage {
	return <-s.messages
}
