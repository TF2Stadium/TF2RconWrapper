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
	rServerCvar         = regexp.MustCompile(`^server_cvar: "(.*)" "(.*)"`)

	rLogFiledClosed = regexp.MustCompile("^Log file closed.")
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

type CvarData struct {
	Variable string
	Value    string
}

const (
	PlayerGlobalMessage = iota
	PlayerTeamMessage
	PlayerChangedClass
	PlayerChangedTeam

	WorldPlayerConnected
	WorldPlayerDisconnected
	WorldGameOver
	ServerCvar

	LogFileClosed = iota
)

type ParsedMsg struct {
	Type int
	Data interface{}
}

var (
	ErrInvalidPacket = errors.New("Invalid Packet")
)

func getSecret(data []byte) (string, int, error) {
	if !(len(data) > 6) {
		return "", 0, ErrInvalidPacket
	}

	if data[4] != 0x53 {
		return "", 0, errors.New("Server trying to send a chat packet without a secret")
	}

	bytes := data[5:]
	var pos int

	for bytes[pos] != 0x20 {
		pos++
		if pos >= len(bytes) {
			return "", 0, ErrInvalidPacket
		}
	}

	secret := string(bytes[:pos-1])
	if pos+1 >= len(data) {
		//No message/time data
		return "", 0, ErrInvalidPacket
	}

	return secret, pos + 1, nil
}

const refTime = "01/02/2006 -  15:04:05"

func parse(text string) LogMessage {
	timeText := text[:21]
	message := text[23:]

	timeObj, _ := time.Parse(refTime, timeText)

	return LogMessage{timeObj, text, ParseLine(message)}
}

func ParseLine(message string) ParsedMsg {
	r := ParsedMsg{Type: -1}
	var m []string

	isPlayerMessage := false
	playerData := PlayerData{}

	switch {
	case rPlayerGlobalMessage.MatchString(message):
		m = rPlayerGlobalMessage.FindStringSubmatch(message)

		isPlayerMessage = true
		playerData.Text = m[5]
		r.Type = PlayerGlobalMessage

	case rPlayerTeamMessage.MatchString(message):
		m = rPlayerTeamMessage.FindStringSubmatch(message)

		isPlayerMessage = true
		playerData.Text = m[5]
		r.Type = PlayerTeamMessage

	case rPlayerChangedClass.MatchString(message):
		m = rPlayerChangedClass.FindStringSubmatch(message)

		isPlayerMessage = true
		playerData.Class = m[5]
		r.Type = PlayerChangedClass

	case rPlayerChangedTeam.MatchString(message):
		m = rPlayerChangedTeam.FindStringSubmatch(message)

		isPlayerMessage = true
		playerData.NewTeam = m[5]
		r.Type = PlayerChangedTeam

	case rPlayerConnected.MatchString(message):
		m = rPlayerConnected.FindStringSubmatch(message)

		isPlayerMessage = true
		r.Type = WorldPlayerConnected

	case rPlayerDisconnected.MatchString(message):
		m = rPlayerDisconnected.FindStringSubmatch(message)

		isPlayerMessage = true
		r.Type = WorldPlayerDisconnected

	// Non-Player Messages
	case rGameOver.MatchString(message):
		r.Type = WorldGameOver

	case rServerCvar.MatchString(message):
		m = rServerCvar.FindStringSubmatch(message)

		r.Type = ServerCvar
		r.Data = CvarData{Variable: m[1], Value: m[2]}

	case rLogFiledClosed.MatchString(message):
		r.Type = LogFileClosed
	}

	// fields used in all matches
	if isPlayerMessage {
		playerData.Username = m[1]
		playerData.SteamId = m[3]
		playerData.UserId = m[2]
		if r.Type != WorldPlayerDisconnected {
			playerData.Team = m[4]
		}
		r.Data = playerData
	}

	return r
}
