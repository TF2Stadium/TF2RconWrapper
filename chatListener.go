package TF2RconWrapper

import (
	"fmt"
	"net"
	"time"
)

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

type RconChatListener struct {
	conn    *net.UDPConn
	channel chan ChatMessage
	exit    chan bool
}

func NewRconChatListener(port int) (*RconChatListener, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+fmt.Sprintf("%v", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	ch := make(chan ChatMessage)
	exit := make(chan bool)

	listener := &RconChatListener{conn, ch, exit}
	go listener.readStrings()

	return listener, nil
}

func (r *RconChatListener) readStrings() {
	buff := make([]byte, 4096)

	for {
		select {
		case <-r.exit:
			return
		default:
			r.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, _, err := r.conn.ReadFromUDP(buff)
			if err != nil {
				if typedErr, ok := err.(*net.OpError); ok && typedErr.Timeout() {
					continue
				}

				fmt.Println("Error receiving server chat data: ", err)
			}

			message := string(buff[0:n])
			r.channel <- proccessMessage(message)
		}
	}
}

func (r *RconChatListener) GetNext() ChatMessage {
	return <-r.channel
}

func (r *RconChatListener) Close() {
	r.exit <- true
	r.conn.Close()
}
