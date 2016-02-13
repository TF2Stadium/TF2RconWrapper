package TF2RconWrapper

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/james4k/rcon"
)

// TF2RconConnection represents a rcon connection to a TF2 server
type TF2RconConnection struct {
	rc   *rcon.RemoteConsole
	host string
}

var (
	ErrUnknownCommand = errors.New("Unknown Command")
	userIDRegex       = regexp.MustCompile(`^#\s+([0-9]+)`)
	nameRegex         = regexp.MustCompile(`"(.*)"`)
	uniqueIDRegex     = regexp.MustCompile(`\[U:\d*:\d*[:1]*\]`)
	IPRegex           = regexp.MustCompile(`\d+\.\d+.\d+.\d+`)
	CVarValueRegex    = regexp.MustCompile(`^"(?:.*?)" = "(.*?)"`)
)

func (c *TF2RconConnection) QueryNoResp(req string) error {
	_, err := c.rc.Write(req)
	return err
}

// Query executes a query and returns the server responses
func (c *TF2RconConnection) Query(req string) (string, error) {
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
		return resp, ErrUnknownCommand
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
	playerString, err := c.Query("status")
	if err != nil {
		return nil, err
	}

	var list []Player
	res := strings.Split(playerString, "\n")
	if len(res) == 0 {
		return list, errors.New("GetPlayers: empty status output")
	}

	for !strings.HasPrefix(res[0], "#") {
		res = res[1:]
	}
	res = res[1:]

	for _, line := range res {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}

		var matches []string

		matches = userIDRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		userID := matches[len(matches)-1]
		loc := nameRegex.FindStringSubmatchIndex(line)
		if loc == nil {
			continue
		}

		name := line[loc[0]:loc[1]]
		matches = uniqueIDRegex.FindStringSubmatch(line[loc[1]:])
		if matches == nil {
			list = append(list, Player{userID, name, "BOT", ""})
			continue
		}

		uniqueID := matches[len(matches)-1]
		ip := IPRegex.FindString(line[loc[1]:])
		list = append(list, Player{userID, name, uniqueID, ip})

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
		c.rc.Close()
		newConnection, _ := rcon.Dial(c.host, password)
		c.rc = newConnection
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
	c.Query(query)
}

// Close closes the connection
func (c *TF2RconConnection) Close() {
	c.rc.Close()
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
	return &TF2RconConnection{rc, address}, nil
}
