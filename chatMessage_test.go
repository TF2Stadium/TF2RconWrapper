package TF2RconWrapper

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var logs []string = []string{
	`"Sk1LL0<2><[U:1:198288660]><Unassigned>" joined team "Red"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "scout"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "soldier"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" say "hello gringos"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" say_team "ufo porno"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" changed role to "sniper"`,
	`"Sk1LL0<2><[U:1:198288660]><Red>" joined team "Blue"`,
	`"Sk1LL0<2><[U:1:198288660]><Blue>" changed role to "medic"`,
}

func TestParse(t *testing.T) {
	for i := range logs {
		m := Parse(logs[i])

		switch i {

		// 0 = changed team
		case 0:
			assert.Equal(t, m.Type, playerChangedTeam)

			assert.Equal(t, m.Data.Team, "Unassigned")
			assert.Equal(t, m.Data.NewTeam, "Red")

			assert.Equal(t, m.Data.Username, "Sk1LL0")
			assert.Equal(t, m.Data.UserId, "2")
			assert.Equal(t, m.Data.SteamId, "[U:1:198288660]")

			// 1 = changed class
		case 1:
			assert.Equal(t, m.Type, playerChangedClass)
			assert.Equal(t, m.Data.Class, "scout")

			// 2 = changed class
		case 2:
			assert.Equal(t, m.Type, playerChangedClass)
			assert.Equal(t, m.Data.Class, "soldier")

			// 3 = global message
		case 3:
			assert.Equal(t, m.Type, playerGlobalMessage)

			assert.Equal(t, m.Data.Team, "Red")
			assert.Equal(t, m.Data.Text, "hello gringos")

			// 4 = team message
		case 4:
			assert.Equal(t, m.Type, playerTeamMessage)
			assert.Equal(t, m.Data.Text, "ufo porno")

			// 5 = changed class
		case 5:
			assert.Equal(t, m.Type, playerChangedClass)
			assert.Equal(t, m.Data.Class, "sniper")

			// 6 = changed team
		case 6:
			assert.Equal(t, m.Type, playerChangedTeam)

			assert.Equal(t, m.Data.Team, "Red")
			assert.Equal(t, m.Data.NewTeam, "Blue")

			// 7 = changed class
		case 7:
			assert.Equal(t, m.Type, playerChangedClass)

			assert.Equal(t, m.Data.Team, "Blue")
			assert.Equal(t, m.Data.Class, "medic")
		}
	}
}
