package TF2RconWrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var logs = []string{
	`"Sk1LL0<2><[U:1:198288660]><Unassigned>" joined team "Red"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "scout"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "soldier"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" say "hello gringos"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" say_team "ufo porno"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "sniper"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" joined team "Blue"`,
	`"Sk1LL0<2><[U:1:198288660]><Blue>" changed role to "medic"`,
	`World triggered "Game_Over" reason "Reached Win Difference Limit"`,
	`server_cvar: "sv_password" "***PROTECTED***"`,
}

func TestParse(t *testing.T) {
	for i := range logs {
		m := ParseLine(logs[i])

		switch i {

		// 0 = changed team
		case 0:
			assert.Equal(t, m.Type, PlayerChangedTeam)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Unassigned")
			assert.Equal(t, playerData.NewTeam, "Red")

			assert.Equal(t, playerData.Username, "Sk1LL0")
			assert.Equal(t, playerData.UserId, "2")
			assert.Equal(t, playerData.SteamId, "[U:1:198288660]")

			// 1 = changed class
		case 1:
			assert.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "scout")

			// 2 = changed class
		case 2:
			assert.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "soldier")

			// 3 = global message
		case 3:
			assert.Equal(t, m.Type, PlayerGlobalMessage)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Red")
			assert.Equal(t, playerData.Text, "hello gringos")

			// 4 = team message
		case 4:
			assert.Equal(t, m.Type, PlayerTeamMessage)
			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Text, "ufo porno")

			// 5 = changed class
		case 5:
			assert.Equal(t, m.Type, PlayerChangedClass)
			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "sniper")

			// 6 = changed team
		case 6:
			assert.Equal(t, m.Type, PlayerChangedTeam)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Red")
			assert.Equal(t, playerData.NewTeam, "Blue")

			// 7 = changed class
		case 7:
			assert.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Blue")
			assert.Equal(t, playerData.Class, "medic")

			// 8 = game over
		case 8:
			assert.Equal(t, m.Type, WorldGameOver)

			// 9 = server cvar
		case 9:
			assert.Equal(t, m.Type, ServerCvar)

			cvarData := m.Data.(CvarData)
			assert.Equal(t, cvarData.Variable, "sv_password")
			assert.Equal(t, cvarData.Value, "***PROTECTED***")
		}
	}
}
