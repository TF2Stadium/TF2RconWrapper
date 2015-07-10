package TF2RconWrapper

import (
	"errors"
	"time"
)

// ChatMessage represents a chat message in a TF2 server, and contains a timestamp and a message.
// The message can be a player message that contains the sender's username, steamid and other info
// or a server message.
type ChatMessage struct {
	Timestamp time.Time
	Message   string
}

func proccessMessage(textBytes []byte) (ChatMessage, string, error) {
	packetType := textBytes[4]

	if packetType != 0x53 {
		return ChatMessage{}, "", errors.New("Server trying to send a chat packet without a secret")
	}

	// drop the header
	textBytes = textBytes[5:]

	pos := 0
	for textBytes[pos] != 0x20 {
		pos++
	}

	secret := string(textBytes[:pos-1])

	textBytes = textBytes[pos+1:]

	text := string(textBytes)

	timeText := text[:21]
	message := text[23:]

	const refTime = "01/02/2006 -  15:04:05"

	timeObj, _ := time.Parse(refTime, timeText)

	return ChatMessage{timeObj, message}, secret, nil
}
