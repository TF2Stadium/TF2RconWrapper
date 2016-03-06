package TF2RconWrapper

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

const (
	// "Username<userId><steamId><Team>"
	// "1<2><3><4>" <- regex group
	logLineStart = `^"(.*)<(\d+)><(\[U:1:\d+\])><(\w+)>" `
	player       = `"(.*)<(\d+)><(\[U:1:\d+\])><(\w+)>"`
	// "5" <- regex group
	logLineEnd = ` "(.*)"`

	logLineStartSpec = `^"(.*)<(\d+)><(\[U:1:\d+\])><(\w*)>" `
)

// regexes used in the parser
var (
	rPlayerGlobalMessage  = regexp.MustCompile(logLineStart + `say` + logLineEnd)
	rPlayerChangedClass   = regexp.MustCompile(logLineStart + `changed role to` + logLineEnd)
	rPlayerTeamMessage    = regexp.MustCompile(logLineStart + `say_team` + logLineEnd)
	rPlayerChangedTeam    = regexp.MustCompile(logLineStart + `joined team` + logLineEnd)
	rPlayerPickedUp       = regexp.MustCompile(logLineStart + `picked up item "(\w+)"(?: \(healing "(\d+)"\))*`)
	rPlayerSpawned        = regexp.MustCompile(logLineStart + `spawned as "(\w+)"`)
	rPlayerKilled         = regexp.MustCompile(logLineStart + `killed ` + player + ` with "(\w+)"(?: \(customkill "(\w+)"\)){0,1} \(attacker_position (".+")\) \(victim_position (".+")\)`)
	rPlayerDamage         = regexp.MustCompile(logLineStart + `triggered "damage" against ` + player + ` \(damage "(\d+)"\)(?: \(realdamage "\d+"\)){0,1} \(weapon "(\w+)"\)(?: \((airshot) "1"\)){0,1}(?: \((headshot) "1"\)){0,1}`)
	rPlayerHeal           = regexp.MustCompile(logLineStart + `triggered "healed" against ` + player + ` \(healing "(\d+)"\)`)
	rPlayerKilledMedic    = regexp.MustCompile(logLineStart + `triggered "medic_death" against ` + player + ` \(healing "(\d+)"\) \(ubercharge "(\d+)"\)`)
	rPlayerUberFinished   = regexp.MustCompile(logLineStart + `triggered "empty_uber"`)
	rPlayerBlockedCapture = regexp.MustCompile(logLineStart + `triggered "captureblocked" \(cp "(\d+)"\) \(cpname "(#\w+)"\) \(position "(.+)"\)`)
	rPlayerConnected      = regexp.MustCompile(logLineStartSpec + `connected, address "\d+.\d+.\d+.\d+\:\d+"`)
	rPlayerDisconnected   = regexp.MustCompile(logLineStartSpec + `disconnected \(reason "(.*)"\)`)

	//Team events
	rTeamPointCapture = regexp.MustCompile(`^Team "(Red|Blue)" triggered "pointcaptured" \(cp "(\d+)"\) \(cpname "(#\w+)"\)`)
	rTeamScoreUpdate  = regexp.MustCompile(`^Team "(Red|Blue)" current score "(\d+)" with "\d+" players`)

	//World events
	rGameOver   = regexp.MustCompile(`^World triggered "Game_Over" reason "(.*)"`)
	rRoundWin   = regexp.MustCompile(`^World triggered "Round_Win" \(winner "(Red|Blue)"\)`)
	rRoundStart = regexp.MustCompile(`^World triggered "Round_Start"`)
	rServerCvar = regexp.MustCompile(`^server_cvar: "(.*)" "(.*)"`)

	rTournamentStarted = regexp.MustCompile(`^Tournament mode started\nBlue Team: \w+\nRed Team: \w`)
	rLogFiledClosed    = regexp.MustCompile("^Log file closed.")
)

const (
	PlayerGlobalMessage = iota
	PlayerTeamMessage
	PlayerChangedClass
	PlayerChangedTeam
	PlayerPickedUpItem
	PlayerSpawned
	PlayerKilled
	PlayerDamaged
	PlayerHealed
	PlayerKilledMedic
	PlayerUberFinished
	PlayerBlockedCapture
	PlayerConnected
	PlayerDisconnected

	TeamPointCapture
	TeamScoreUpdate

	WorldGameOver
	WorldRoundWin
	WorldRoundStart
	ServerCvar

	TournamentStarted
	LogFileClosed
)

//LogMessage represents a log message in a TF2 server, and contains a timestamp
//and a message. The message can be a player message that contains the sender's
//username, steamid and other info or a server message.
type LogMessage struct {
	Timestamp time.Time
	Message   string
	Parsed    ParsedMsg
}

type TeamData struct {
	CPData
	Team  string `json:"team"`
	Score string `json:"score"`
}

type CPData struct {
	CP     string `json:"cp"`
	CPName string `json:"cpname"`
}
type ItemPickup struct {
	PlayerData PlayerData `json:"player"`
	Item       string     `json:"item"`
	Healing    int        `json:"healing"`
}

type PlayerTrigger struct {
	//player1 triggered "Foo" against player2
	Player1 PlayerData `json:"player1"`
	Player2 PlayerData `json:"player2"`
}

type PlayerKill struct {
	PlayerTrigger
	Weapon     string `json:"weapon"`
	CustomKill string `json:"customkill"`
}

type PlayerDamage struct {
	PlayerTrigger
	Damage   int    `json:"damage"`
	Weapon   string `json:"weapon"`
	Airshot  bool   `json:"airshot"`
	Headshot bool   `json:"headshot"`
}

type PlayerHeal struct {
	PlayerTrigger
	Healed int `json:"healed"` // health gained
}

//When a player says something in the global chat, or when they join the game
type PlayerData struct {
	Username string `json:"username"`

	SteamId string `json:"steamid"`
	UserId  string `json:"userid"`

	Team    string `json:"team"`
	NewTeam string `json:"newteam"`

	Text  string `json:"text"`
	Class string `json:"class"`
}

type CvarData struct {
	Variable string
	Value    string
}

type ParsedMsg struct {
	Type int
	Data interface{}
}

var (
	//ErrInvalidPacket represents an error when an invalid packet is found
	ErrInvalidPacket = errors.New("invalid packet")
)

func (p *ParsedMsg) CallHandler(handler *EventListener) {
	var eventhandler interface{}
	var in []interface{}

	switch p.Type {
	case PlayerGlobalMessage:
		d := p.Data.(PlayerData)
		eventhandler = handler.PlayerGlobalMessage
		in = []interface{}{d, d.Text}
	case PlayerTeamMessage:
		d := p.Data.(PlayerData)
		eventhandler = handler.PlayerTeamMessage
		in = []interface{}{d, d.Text}
	case PlayerChangedClass:
		d := p.Data.(PlayerData)
		eventhandler = handler.PlayerClassChanged
		in = []interface{}{d, d.Text}
	case PlayerPickedUpItem:
		eventhandler = handler.PlayerItemPickup
		in = []interface{}{p.Data.(ItemPickup)}
	case PlayerSpawned:
		eventhandler = handler.PlayerSpawned
		d := p.Data.(PlayerData)
		in = []interface{}{d, d.Class}
	case PlayerChangedTeam:
		d := p.Data.(PlayerData)
		in = []interface{}{d, d.NewTeam}
		eventhandler = handler.PlayerTeamChange
	case PlayerConnected:
		in = []interface{}{p.Data.(PlayerData)}
		eventhandler = handler.PlayerConnected
	case PlayerKilled:
		in = []interface{}{p.Data.(PlayerKill)}
		eventhandler = handler.PlayerKilled
	case PlayerDamaged:
		in = []interface{}{p.Data.(PlayerDamage)}
		eventhandler = handler.PlayerDamaged
	case PlayerHealed:
		in = []interface{}{p.Data.(PlayerHeal)}
		eventhandler = handler.PlayerHealed
	case PlayerKilledMedic:
		in = []interface{}{p.Data.(PlayerTrigger)}
		eventhandler = handler.PlayerKilledMedic
	case PlayerUberFinished:
		in = []interface{}{p.Data.(PlayerData)}
		eventhandler = handler.PlayerUberFinished
	case PlayerBlockedCapture:
		arr := p.Data.([]interface{})
		in = []interface{}{arr[0], arr[1]}
		eventhandler = handler.PlayerBlockedCapture
	case TeamPointCapture:
		in = []interface{}{p.Data.(TeamData)}
		eventhandler = handler.TeamPointCapture
	case TeamScoreUpdate:
		in = []interface{}{p.Data.(TeamData)}
		eventhandler = handler.TeamScoreUpdate
	case WorldRoundWin:
		in = []interface{}{p.Data.(string)}
		eventhandler = handler.WorldRoundWin
	//case WorldRoundStart:
	case PlayerDisconnected:
		in = []interface{}{p.Data.(PlayerData)}
		eventhandler = handler.PlayerDisconnected
	case WorldGameOver:
		eventhandler = handler.GameOver
	case ServerCvar:
		in = []interface{}{p.Data.(CvarData).Value, p.Data.(CvarData).Variable}
		eventhandler = handler.CVarChange
	case LogFileClosed:
		eventhandler = handler.LogFileClosed
	case TournamentStarted:
		eventhandler = handler.TournamentStarted
	default:
		return
	}

	var args []reflect.Value
	for _, arg := range in {
		args = append(args, reflect.ValueOf(arg))
	}

	method := reflect.ValueOf(eventhandler)
	if !method.IsValid() || method.IsNil() {
		return
	}
	method.Call(args)
}

func getSecret(data []byte) (string, int, error) {
	if !(len(data) > 6) {
		return "", 0, ErrInvalidPacket
	}

	if data[4] != 0x53 {
		return "", 0, errors.New("Server trying to send a chat packet without a secret")
	}

	bytes := data[5:]
	pos := 5

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
	timeText := text[5:26]
	message := text[28:]

	timeObj, _ := time.Parse(refTime, timeText)
	return LogMessage{timeObj, text, ParseLine(message)}
}

func getPlayerData(matches []string, from int, includeTeam bool) PlayerData {
	d := PlayerData{
		Username: matches[from+0],
		UserId:   matches[from+1],
		SteamId:  matches[from+2],
	}

	if includeTeam {
		d.Team = matches[from+3]
	}

	return d
}

func ParseLine(message string) ParsedMsg {
	r := ParsedMsg{Type: -1}

	switch {
	case rPlayerGlobalMessage.MatchString(message):
		m := rPlayerGlobalMessage.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, true)
		playerData.Text = m[5]
		r.Data = playerData
		r.Type = PlayerGlobalMessage

	case rPlayerTeamMessage.MatchString(message):
		m := rPlayerTeamMessage.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, true)
		playerData.Text = m[5]
		r.Data = playerData
		r.Type = PlayerTeamMessage

	case rPlayerChangedClass.MatchString(message):
		m := rPlayerChangedClass.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, true)
		playerData.Class = m[5]
		r.Data = playerData
		r.Type = PlayerChangedClass

	case rPlayerChangedTeam.MatchString(message):
		m := rPlayerChangedTeam.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, true)
		playerData.NewTeam = m[5]
		r.Data = playerData
		r.Type = PlayerChangedTeam

	case rPlayerSpawned.MatchString(message):
		m := rPlayerSpawned.FindStringSubmatch(message)
		playerData := getPlayerData(m, 1, true)
		playerData.Class = m[5]
		r.Data = playerData
		r.Type = PlayerSpawned

	case rPlayerPickedUp.MatchString(message):
		m := rPlayerPickedUp.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, true)
		pickup := ItemPickup{
			PlayerData: playerData,
			Item:       m[5],
		}

		if len(m) == 7 {
			pickup.Healing, _ = strconv.Atoi(m[6])
		}

		r.Data = pickup
		r.Type = PlayerPickedUpItem

	case rPlayerKilled.MatchString(message):
		m := rPlayerKilled.FindStringSubmatch(message)

		kill := PlayerKill{
			PlayerTrigger: PlayerTrigger{
				Player1: getPlayerData(m, 1, true),
				Player2: getPlayerData(m, 5, true),
			},
		}

		kill.Weapon = m[9]
		if len(m) > 10 {
			kill.CustomKill = m[10]
		}

		r.Data = kill
		r.Type = PlayerKilled

	case rPlayerDamage.MatchString(message):
		m := rPlayerDamage.FindStringSubmatch(message)

		dmg, _ := strconv.Atoi(m[9])
		damage := PlayerDamage{
			PlayerTrigger: PlayerTrigger{
				Player1: getPlayerData(m, 1, true), // ends at m[4]
				Player2: getPlayerData(m, 5, true), // ends at m[8]
			},

			Damage:   dmg,
			Weapon:   m[10],
			Airshot:  m[11] == "airshot",
			Headshot: m[12] == "headshot",
		}

		r.Data = damage
		r.Type = PlayerDamaged

	case rPlayerHeal.MatchString(message):
		m := rPlayerHeal.FindStringSubmatch(message)

		healing, _ := strconv.Atoi(m[9])
		heal := PlayerHeal{
			PlayerTrigger: PlayerTrigger{
				Player1: getPlayerData(m, 1, true), // end at m[4]
				Player2: getPlayerData(m, 5, true), // ends at m[8]
			},
			Healed: healing,
		}
		r.Data = heal
		r.Type = PlayerHealed

	case rPlayerKilledMedic.MatchString(message):
		m := rPlayerKilledMedic.FindStringSubmatch(message)

		r.Data = PlayerTrigger{
			Player1: getPlayerData(m, 1, true),
			Player2: getPlayerData(m, 5, true),
		}
		r.Type = PlayerKilledMedic

	case rPlayerUberFinished.MatchString(message):
		m := rPlayerUberFinished.FindStringSubmatch(message)

		r.Data = getPlayerData(m, 1, true)
		r.Type = PlayerUberFinished

	case rPlayerBlockedCapture.MatchString(message):
		m := rPlayerBlockedCapture.FindStringSubmatch(message)

		r.Data = []interface{}{CPData{m[5], m[6]}, getPlayerData(m, 1, true)}

		r.Type = PlayerBlockedCapture

	case rPlayerConnected.MatchString(message):
		m := rPlayerConnected.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, false)
		r.Data = playerData
		r.Type = PlayerConnected

	case rPlayerDisconnected.MatchString(message):
		m := rPlayerDisconnected.FindStringSubmatch(message)

		playerData := getPlayerData(m, 1, false)
		r.Data = playerData
		r.Type = PlayerDisconnected

	// Non-Player Messages
	case rGameOver.MatchString(message):
		r.Type = WorldGameOver

	case rRoundWin.MatchString(message):
		m := rRoundWin.FindStringSubmatch(message)
		r.Data = m[1]
		r.Type = WorldRoundWin

	case rServerCvar.MatchString(message):
		m := rServerCvar.FindStringSubmatch(message)

		r.Type = ServerCvar
		r.Data = CvarData{Variable: m[1], Value: m[2]}

	case rLogFiledClosed.MatchString(message):
		r.Type = LogFileClosed
	case rTournamentStarted.MatchString(message):
		r.Type = TournamentStarted
	case rTeamPointCapture.MatchString(message):
		m := rTeamPointCapture.FindStringSubmatch(message)
		team := TeamData{
			Team: m[1],
			CPData: CPData{
				CP:     m[2],
				CPName: m[3],
			}}
		r.Data = team
		r.Type = TeamPointCapture
	case rTeamScoreUpdate.MatchString(message):
		m := rTeamScoreUpdate.FindStringSubmatch(message)
		team := TeamData{
			Team:  m[1],
			Score: m[2],
		}
		r.Data = team
		r.Type = TeamScoreUpdate
	}

	return r
}
