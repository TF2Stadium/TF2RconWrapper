package TF2RconWrapper

import (
	"errors"
	"regexp"
	"time"
)

const (
	// "Username<userId><steamId><Team>"
	// "1<2><3><4>" <- regex group
	logLineStart = `^"(.*)<(\d+)><(\[U:1:\d+\])><(\w+)>" `
	// "5" <- regex group
	logLineEnd = ` "(.*)"`

	logLineStartSpec = `^"(.*)<(\d+)><(\[U:1:\d+\])><(\w*)>" `
)

// regexes used in the parser
var (
	rPlayerGlobalMessage = regexp.MustCompile(logLineStart + `say` + logLineEnd)
	rPlayerChangedClass  = regexp.MustCompile(logLineStart + `changed role to` + logLineEnd)
	rPlayerTeamMessage   = regexp.MustCompile(logLineStart + `say_team` + logLineEnd)
	rPlayerChangedTeam   = regexp.MustCompile(logLineStart + `joined team` + logLineEnd)

	//Global Messages
	rPlayerConnected    = regexp.MustCompile(logLineStartSpec + `connected, address "\d+.\d+.\d+.\d+\:\d+"`)
	rPlayerDisconnected = regexp.MustCompile(logLineStartSpec + `disconnected \(reason "(.*)"\)`)
	rGameOver           = regexp.MustCompile(`^World triggered "Game_Over" reason "(.*)"`)
)

//LogMessage represents a log message in a TF2 server, and contains a timestamp
//and a message. The message can be a player message that contains the sender's
//username, steamid and other info or a server message.
type LogMessage struct {
	Timestamp time.Time
	Message   string
	Parsed    ParsedMsg
}

//When a player says something in the global chat, or when they join the game
type PlayerData struct {
	Username string

	SteamId string
	UserId  string

	Team    string
	NewTeam string

	Text  string
	Class string
}

const (
	PlayerGlobalMessage = iota
	PlayerTeamMessage   = iota
	PlayerChangedClass  = iota
	PlayerChangedTeam   = iota

	WorldPlayerConnected    = iota
	WorldPlayerDisconnected = iota
	WorldGameOver           = iota
)

type ParsedMsg struct {
	Type int
	Data PlayerData
}

func proccessMessage(textBytes []byte) (LogMessage, string, error) {
	packetType := textBytes[4]

	if packetType != 0x53 {
		return LogMessage{}, "", errors.New("Server trying to send a chat packet without a secret")
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

	m := Parse(message)

	return LogMessage{timeObj, message, m}, secret, nil
}

func Parse(message string) ParsedMsg {
	r := ParsedMsg{Type: -1}
	var m []string

	switch {
	case rPlayerGlobalMessage.MatchString(message):
		m = rPlayerGlobalMessage.FindStringSubmatch(message)

		r.Data.Text = m[5]
		r.Type = PlayerGlobalMessage

	case rPlayerTeamMessage.MatchString(message):
		m = rPlayerTeamMessage.FindStringSubmatch(message)

		r.Data.Text = m[5]
		r.Type = PlayerTeamMessage

	case rPlayerChangedClass.MatchString(message):
		m = rPlayerChangedClass.FindStringSubmatch(message)

		r.Data.Class = m[5]
		r.Type = PlayerChangedClass

	case rPlayerChangedTeam.MatchString(message):
		m = rPlayerChangedTeam.FindStringSubmatch(message)

		r.Data.NewTeam = m[5]
		r.Type = PlayerChangedTeam

	case rGameOver.MatchString(message):
		r.Type = WorldGameOver

	case rPlayerConnected.MatchString(message):
		m = rPlayerConnected.FindStringSubmatch(message)

		r.Type = WorldPlayerConnected

	case rPlayerDisconnected.MatchString(message):
		m = rPlayerDisconnected.FindStringSubmatch(message)

		r.Type = WorldPlayerDisconnected
	}

	// fields used in all matches
	if r.Type != -1 && r.Type != WorldGameOver {
		r.Data.Username = m[1]
		r.Data.SteamId = m[3]
		r.Data.UserId = m[2]
		if r.Type != WorldPlayerDisconnected {
			r.Data.Team = m[4]
		}
	}

	return r
}
