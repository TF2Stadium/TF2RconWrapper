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
	Channel chan ChatMessage
	exit    chan int
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
	exit := make(chan int)

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
			n, _, err := r.conn.ReadFromUDP(buff)
			if err != nil {
				fmt.Println("Error receiving server chat data: ", err)
			}

			message := string(buff[0:n])
			r.Channel <- proccessMessage(message)
		}
	}
}

func (r *RconChatListener) Close() {
	fmt.Print("started close")
	r.exit <- 1
	r.conn.Close()
	fmt.Print("ended")
}
