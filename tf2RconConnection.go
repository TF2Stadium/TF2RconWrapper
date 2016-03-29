package TF2RconWrapper

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/james4k/rcon"
)

// TF2RconConnection represents a rcon connection to a TF2 server
type TF2RconConnection struct {
	rcLock sync.RWMutex
	rc     *rcon.RemoteConsole

	host     string
	password string
}

var (
	ErrUnknownCommand = errors.New("Unknown Command")
	CVarValueRegex    = regexp.MustCompile(`^"(?:.*?)" = "(.*?)"`)
	//# userid name                uniqueid            connected ping loss state  adr
	rePlayerInfo = regexp.MustCompile(`^#\s+(\d+)\s+"(.+)"\s+(\[U:1:\d+\])\s+\d+:\d+\s+\d+\s+\d+\s+\w+\s+(\d+\.+\d+\.\d+\.\d+:\d+)`)
)

type UnknownCommand string

func (c UnknownCommand) Error() string {
	return "unknown command: " + string(c)
}

func (c *TF2RconConnection) QueryNoResp(req string) error {
	if c.rc == nil {
		return errors.New("RCON connection is nil")
	}

	c.rcLock.RLock()
	defer c.rcLock.RUnlock()

	_, err := c.rc.Write(req)
	return err
}

// Query executes a query and returns the server responses
func (c *TF2RconConnection) Query(req string) (string, error) {
	c.rcLock.RLock()
	defer c.rcLock.RUnlock()

	if c.rc == nil {
		return "", errors.New("RCON connection is nil")
	}

	reqID, reqErr := c.rc.Write(req)
	if reqErr != nil {
		// log.Println(reqErr)
		return "", reqErr
	}

	resp, respID, respErr := c.rc.Read()
	if respErr != nil {
		// log.Println(respErr)
		return "", respErr
	}

	counter := 10
	// retry 10 times
	for {
		if reqID == respID {
			break
		} else if counter < 0 {
			return "", errors.New("Couldn't get a response.")
		} else {
			counter--
			resp, respID, respErr = c.rc.Read()
			if respErr != nil {
				// log.Println(respErr)
				return "", reqErr
			}
		}
	}

	if strings.HasPrefix(resp, "Unknown command") {
		return resp, UnknownCommand(req)
	}

	return resp, nil
}

func (c *TF2RconConnection) GetConVar(cvar string) (string, error) {
	raw, err := c.Query(cvar)

	if err != nil {
		return "", err
	}

	// Querying just a variable's name sends back a message like the
	// following:
	//
	// "cvar_name" = "current value" ( def. "default value" )
	//  var flags like notify replicated
	//  - short description of cvar

	firstLine := strings.Split(raw, "\n")[0]
	matches := CVarValueRegex.FindStringSubmatch(firstLine)
	if len(matches) != 2 {
		return "", errors.New("Unknown cvar.")
	}

	return matches[1], nil
}

func (c *TF2RconConnection) SetConVar(cvar string, val string) (string, error) {
	return c.Query(fmt.Sprintf("%s \"%s\"", cvar, val))
}

// GetPlayers returns a list of players in the server. Includes bots.
func (c *TF2RconConnection) GetPlayers() ([]Player, error) {
	statusString, err := c.Query("status")
	if err != nil {
		return nil, err
	}

	index := strings.Index(statusString, "#")
	i := 0
	for index == -1 {
		statusString, _ = c.Query("status")
		index = strings.Index(statusString, "#")
		i++
		if i == 5 {
			return nil, errors.New("Couldn't get output of status")
		}
	}

	users := strings.Split(statusString[index:], "\n")
	var list []Player
	for _, userString := range users {
		if !rePlayerInfo.MatchString(userString) {
			continue
		}
		matches := rePlayerInfo.FindStringSubmatch(userString)
		player := Player{
			UserID:   matches[1],
			Username: matches[2],
			SteamID:  matches[3],
			Ip:       matches[4],
		}
		list = append(list, player)
	}

	return list, nil
}

// KickPlayer kicks a player
func (c *TF2RconConnection) KickPlayer(p Player, message string) error {
	return c.KickPlayerID(p.UserID, message)
}

// Kicks a player with the given player ID
func (c *TF2RconConnection) KickPlayerID(userID string, message string) error {
	query := fmt.Sprintf("kickid %s %s", userID, message)
	_, err := c.Query(query)
	return err
}

// BanPlayer bans a player
func (c *TF2RconConnection) BanPlayer(minutes int, p Player, message string) error {
	query := "banid " + fmt.Sprintf("%v", minutes) + " " + p.UserID
	if message != "" {
		query += " \"" + message + "\""
	}
	_, err := c.Query(query)
	return err
}

// UnbanPlayer unbans a player
func (c *TF2RconConnection) UnbanPlayer(p Player) error {
	query := "unbanid " + p.UserID
	_, err := c.Query(query)
	return err
}

// Say sends a message to the TF2 server chat
func (c *TF2RconConnection) Say(message string) error {
	query := "say " + message
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) Sayf(format string, a ...interface{}) error {
	err := c.Say(fmt.Sprintf(format, a...))
	return err
}

// ChangeRconPassword changes the rcon password and updates the current connection
// to use the new password
func (c *TF2RconConnection) ChangeRconPassword(password string) error {
	_, err := c.SetConVar("rcon_password", password)

	if err == nil {
		err = c.Reconnect(1 * time.Minute)
	}

	return err
}

// ChangeMap changes the map
func (c *TF2RconConnection) ChangeMap(mapname string) error {
	query := "changelevel \"" + mapname + "\""
	res, err := c.Query(query)
	if res != "" {
		return errors.New("Map not found.")
	}
	return err
}

// ChangeServerPassword changes the server password
func (c *TF2RconConnection) ChangeServerPassword(password string) error {
	_, err := c.SetConVar("sv_password", password)
	return err
}

// GetServerPassword returns the server password
func (c *TF2RconConnection) GetServerPassword() (string, error) {
	return c.GetConVar("sv_password")
}

func (c *TF2RconConnection) AddTag(newTag string) error {
	tags, err := c.GetConVar("sv_tags")
	if err != nil {
		return err
	}

	// Source servers don't auto-remove duplicate tags, and noone
	tagExists := false
	for _, tag := range strings.Split(tags, ",") {
		if tag == newTag {
			tagExists = true
			break
		}
	}

	if !tagExists {
		newTags := strings.Join([]string{tags, newTag}, ",")
		_, err := c.SetConVar("sv_tags", newTags)
		return err
	}

	return nil
}

func (c *TF2RconConnection) RemoveTag(tagName string) error {
	tags, err := c.GetConVar("sv_tags")
	if err != nil {
		return err
	}

	if strings.Contains(tags, tagName) {
		// Replace all instances of the given tagName. This may leave
		// duplicated or trailing commas in the sv_tags string; however
		// Source servers clean up the value of sv_tags to remove those
		// anyways
		_, err := c.SetConVar("sv_tags", strings.Replace(tags, tagName, "", -1))
		return err
	}

	return nil
}

// RedirectLogs send the logaddress_add command
func (c *TF2RconConnection) RedirectLogs(addr string) error {
	query := "logaddress_add " + addr
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) StopLogRedirection(addr string) {
	query := fmt.Sprintf("logaddress_del %s", addr)
	c.QueryNoResp(query)
}

// Close closes the connection
func (c *TF2RconConnection) Close() {
	c.rcLock.Lock()
	if c.rc != nil {
		c.rc.Close()
	}
	c.rcLock.Unlock()
}

// ExecConfig accepts a string and executes its lines one by one. Assumes
// UNiX line endings
func (c *TF2RconConnection) ExecConfig(config string) error {
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		_, err := c.Query(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewTF2RconConnection builds a new TF2RconConnection to a server at address ("ip:port") using
// a rcon_password password
func NewTF2RconConnection(address, password string) (*TF2RconConnection, error) {
	rc, err := rcon.Dial(address, password)
	if err != nil {
		return nil, err
	}

	return &TF2RconConnection{
		rc:       rc,
		host:     address,
		password: password}, nil
}

func (c *TF2RconConnection) Reconnect(duration time.Duration) error {
	c.Close()

	c.rcLock.Lock()
	defer c.rcLock.Unlock()

	if c.rc == nil {
		return errors.New("RCON connection is nil")
	}

	now := time.Now()
	var err error

	for time.Since(now) <= duration {
		c.rc, err = rcon.Dial(c.host, c.password)
		if err == nil {
			return nil
		}
	}

	return err
}
