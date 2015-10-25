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
	UnknownCommandError = errors.New("Unknown Command")
	userIDRegex         = regexp.MustCompile(`^#\s+([0-9]+)`)
	nameRegex           = regexp.MustCompile(`\"(.*)\"`)
	uniqueIDRegex       = regexp.MustCompile(`\[U:\d*:\d*[:1]*\]`)
	IPRegex             = regexp.MustCompile(`\d+\.\d+.\d+.\d+`)
)

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
		return "", UnknownCommandError
	}

	return resp, nil
}

// GetPlayers returns a list of players in the server. Includes bots.
func (c *TF2RconConnection) GetPlayers() ([]Player, error) {
	playerString, err := c.Query("status")
	if err != nil {
		return nil, err
	}

	res := strings.Split(playerString, "\n")
	for !strings.HasPrefix(res[0], "#") {
		res = res[1:]
	}
	res = res[1:]
	var list []Player
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
	query := "kickid " + p.UserID
	if message != "" {
		query += " \"" + message + "\""
	}
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

// ChangeRconPassword changes the rcon password and updates the current connection
// to use the new password
func (c *TF2RconConnection) ChangeRconPassword(password string) error {
	query := "rcon_password \"" + password + "\""
	_, err := c.Query(query)

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
		return errors.New(res)
	}
	return err
}

// ChangeServerPassword changes the server password
func (c *TF2RconConnection) ChangeServerPassword(password string) error {
	query := fmt.Sprintf("sv_password %s", password)
	_, err := c.Query(query)
	return err
}

// RedirectLogs send the logaddress_add command
func (c *TF2RconConnection) RedirectLogs(ip string, port string) error {
	query := "logaddress_add " + ip + ":" + port
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) StopLogRedirection(localip string, port string) {
	query := fmt.Sprintf("logaddress_del %s:%s", localip, port)
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
