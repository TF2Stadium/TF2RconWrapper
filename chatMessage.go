package TF2RconWrapper

import (
	"errors"
	"regexp"
	"time"
)

// "Username<userId><steamId><Team>"
// "1<2><3><4>" <- regex group
const logLineStart = `^"(.*)<(\d+)><(\[U:1:\d+\])><(\w+)>" `

// "5" <- regex group
const logLineEnd = ` "(.*)"`

// regexes used in the parser
var rPlayerGlobalMessage *regexp.Regexp
var rPlayerChangedClass *regexp.Regexp
var rPlayerTeamMessage *regexp.Regexp
var rPlayerChangedTeam *regexp.Regexp
var rGameOver *regexp.Regexp

var compiledRegexes bool = false

// ChatMessage represents a chat message in a TF2 server, and contains a timestamp and a message.
// The message can be a player message that contains the sender's username, steamid and other info
// or a server message.
type ChatMessage struct {
	Timestamp time.Time
	Message   string
	Parsed    ParsedMsg
}

// when a player say something in global chat
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
	WorldGameOver       = iota
)

type ParsedMsg struct {
	Type int
	Data PlayerData
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

	m := Parse(message)

	return ChatMessage{timeObj, message, m}, secret, nil
}

func compileRegexes() {
	rPlayerGlobalMessage, _ = regexp.Compile(logLineStart + `say` + logLineEnd)
	rPlayerChangedClass, _ = regexp.Compile(logLineStart + `changed role to` + logLineEnd)
	rPlayerTeamMessage, _ = regexp.Compile(logLineStart + `say_team` + logLineEnd)
	rPlayerChangedTeam, _ = regexp.Compile(logLineStart + `joined team` + logLineEnd)
	rGameOver, _ = regexp.Compile(`^World triggered "Game_Over" reason "(.*)"`)

	compiledRegexes = true
}

func Parse(message string) ParsedMsg {
	r := ParsedMsg{Type: -1}
	var m []string

	// we don't need to compile them everytime...
	if !compiledRegexes {
		compileRegexes()
	}

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
	}

	// fields used in all matches
	if r.Type != -1 && r.Type != WorldGameOver {
		r.Data.Username = m[1]
		r.Data.SteamId = m[3]
		r.Data.UserId = m[2]
		r.Data.Team = m[4]
	}

	return r
}
