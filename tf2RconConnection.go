package TF2RconWrapper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/james4k/rcon"
)

// TF2RconConnection represents a rcon connection to a TF2 server
type TF2RconConnection struct {
	rc   *rcon.RemoteConsole
	host string
}

// Query executes a query and returns the server responses
func (c *TF2RconConnection) Query(req string) (string, error) {
	reqID, reqErr := c.rc.Write(req)
	if reqErr != nil {
		fmt.Print(reqErr)
		return "", reqErr
	}

	resp, respID, respErr := c.rc.Read()
	if respErr != nil {
		fmt.Print(reqErr)
		return "", reqErr
	}

	// retry until you get a response
	for {
		if reqID == respID {
			break
		} else {
			resp, respID, respErr = c.rc.Read()
			if respErr != nil {
				fmt.Print(reqErr)
				return "", reqErr
			}
		}
	}

	return resp, nil
}

// GetPlayers returns a list of players in the server. Includes bots.
func (c *TF2RconConnection) GetPlayers() []Player {
	playerString, _ := c.Query("status")
	res := strings.Split(playerString, "\n")
	for !strings.HasPrefix(res[0], "#") {
		res = res[1:]
	}
	res = res[1:]
	var list []Player
	for _, elem := range res {
		if elem == "" {
			continue
		}
		elems := strings.Fields(elem)[1:]
		userID := elems[0]
		name := elems[1]
		name = name[1 : len(name)-1]
		uniqueID := elems[2]
		if uniqueID == "BOT" {
			list = append(list, Player{userID, name, uniqueID, 0, "active", ""})
		} else {
			ping, _ := strconv.Atoi(elems[4])
			state := elems[6]
			ip := elems[7]
			list = append(list, Player{userID, name, uniqueID, ping, state, ip})
		}
	}
	return list
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

// ChangeServerPassword changes the server password
func (c *TF2RconConnection) ChangeServerPassword(password string) error {
	query := "sv_password \"" + password + "\""
	_, err := c.Query(query)
	return err
}

// RedirectLogs send the logaddress_add command
func (c *TF2RconConnection) RedirectLogs(ip string, port string) error {
	query := "logaddress_add " + ip + ":" + port
	_, err := c.Query(query)
	return err
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
