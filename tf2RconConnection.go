package TF2RconWrapper

import (
	"fmt"
	"strings"

	"github.com/james4k/rcon"
)

type TF2RconConnection struct {
	rc *rcon.RemoteConsole
}

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

func (c *TF2RconConnection) GetPlayers() []Player {
	playerString, _ := c.Query("users")
	res := strings.Split(playerString, "\n")
	res = res[1 : len(res)-2]
	var list []Player
	for _, elem := range res {
		data := strings.Split(elem, ":")
		list = append(list, Player{data[1], data[2], data[0], ""})
	}
	return list
}

func (c *TF2RconConnection) KickPlayer(p Player, message string) error {
	query := "kickid " + p.PlayerID
	if message != "" {
		query += " \"" + message + "\""
	}
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) BanPlayer(minutes int, p Player, message string) error {
	query := "banid " + fmt.Sprintf("%v", minutes) + " " + p.PlayerID
	if message != "" {
		query += " \"" + message + "\""
	}
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) UnbanPlayer(p Player) error {
	query := "unbanid " + p.PlayerID
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) Say(message string) error {
	query := "say \"" + message + "\""
	_, err := c.Query(query)
	return err
}

func (c *TF2RconConnection) ChangePassword(password string) error {
	query := "sv_password \"" + password + "\""
	_, err := c.Query(query)
	return err
}

func NewTF2RconConnection(address, password string) (*TF2RconConnection, error) {
	rc, err := rcon.Dial(address, password)
	if err != nil {
		return nil, err
	}
	return &TF2RconConnection{rc}, nil
}
