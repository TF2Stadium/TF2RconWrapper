package TF2RconWrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	`"Sk1LL0<2><[U:1:198288660]><Blue>" picked up item "medkit_small" (healing "17")`,
	`"Tedstur<9><[U:1:98355052]><Red>" killed "Lyreix | TF2Stadium.com<4><[U:1:56108026]><Blue>" with "scattergun" (attacker_position "-2310 -303 256") (victim_position "-2180 -474 256")`,
	`"Tedstur<9><[U:1:98355052]><Red>" killed "Lyreix | TF2Stadium.com<4><[U:1:56108026]><Blue>" with "awper_hand" (customkill "headshot") (attacker_position "-673 1008 384") (victim_position "-754 802 391")`,
	`"≫HarZe<3><[U:1:40572775]><Blue>" triggered "damage" against "beastie<5><[U:1:28701225]><Red>" (damage "100") (realdamage "88") (weapon "iron_bomber")`,
	`"emkay lft<8><[U:1:64912509]><Red>" triggered "damage" against "≫HarZe<3><[U:1:40572775]><Blue>" (damage "43") (weapon "tf_projectile_rocket") (airshot "1")`,
	`"Slappy™<11><[U:1:56973094]><Blue>" triggered "healed" against "mu<12><[U:1:33573908]><Blue>" (healing "61")`,
	`"Lyreix | TF2Stadium.com<4><[U:1:56108026]><Blue>" triggered "medic_death" against "crab_f ring plz<6><[U:1:84999165]><Red>" (healing "802") (ubercharge "0")`,
	`World triggered "Round_Win" (winner "Blue")`,
	`"≫HarZe<3><[U:1:40572775]><>" connected, address "0.0.0.0:27005"`,
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
			require.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "scout")

			// 2 = changed class
		case 2:
			require.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "soldier")

			// 3 = global message
		case 3:
			require.Equal(t, m.Type, PlayerGlobalMessage)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Red")
			assert.Equal(t, playerData.Text, "hello gringos")

			// 4 = team message
		case 4:
			require.Equal(t, m.Type, PlayerTeamMessage)
			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Text, "ufo porno")

			// 5 = changed class
		case 5:
			require.Equal(t, m.Type, PlayerChangedClass)
			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Class, "sniper")

			// 6 = changed team
		case 6:
			require.Equal(t, m.Type, PlayerChangedTeam)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Red")
			assert.Equal(t, playerData.NewTeam, "Blue")

			// 7 = changed class
		case 7:
			require.Equal(t, m.Type, PlayerChangedClass)

			playerData := m.Data.(PlayerData)
			assert.Equal(t, playerData.Team, "Blue")
			assert.Equal(t, playerData.Class, "medic")

			// 8 = game over
		case 8:
			require.Equal(t, m.Type, WorldGameOver)

			// 9 = server cvar
		case 9:
			require.Equal(t, m.Type, ServerCvar)

			cvarData := m.Data.(CvarData)
			assert.Equal(t, cvarData.Variable, "sv_password")
			assert.Equal(t, cvarData.Value, "***PROTECTED***")
		case 10:
			require.Equal(t, m.Type, PlayerPickedUpItem)

			pickup := m.Data.(ItemPickup)
			assert.Equal(t, pickup.Item, "medkit_small")
			assert.Equal(t, pickup.Healing, 17)
		case 11:
			require.Equal(t, m.Type, PlayerKilled)
			kill := m.Data.(PlayerKill)
			assert.Equal(t, kill.Weapon, "scattergun")
		case 12:
			require.Equal(t, m.Type, PlayerKilled)
			kill := m.Data.(PlayerKill)
			assert.Equal(t, kill.CustomKill, "headshot")
			assert.Equal(t, kill.Weapon, "awper_hand")
		case 13:
			require.Equal(t, m.Type, PlayerDamaged)
			damage := m.Data.(PlayerDamage)
			assert.Equal(t, damage.Player2, PlayerData{
				Username: "beastie",
				UserId:   "5",
				SteamId:  "[U:1:28701225]",
				Team:     "Red",
			})
			assert.Equal(t, damage.Damage, 100)
			assert.Equal(t, damage.Weapon, "iron_bomber")
		case 14:
			require.Equal(t, m.Type, PlayerDamaged)
			damage := m.Data.(PlayerDamage)
			assert.Equal(t, damage.Player2, PlayerData{
				Username: "≫HarZe",
				UserId:   "3",
				SteamId:  "[U:1:40572775]",
				Team:     "Blue",
			})
			assert.Equal(t, damage.Damage, 43)
			assert.Equal(t, damage.Weapon, "tf_projectile_rocket")
			assert.True(t, damage.Airshot)
		case 15:
			require.Equal(t, m.Type, PlayerHealed)
			assert.Equal(t, m.Data.(PlayerHeal).Healed, 61)
		case 16:
			require.Equal(t, m.Type, PlayerKilledMedic)
		case 17:
			require.Equal(t, m.Type, WorldRoundWin)
			assert.Equal(t, m.Data.(string), "Blue")
		case 18:
			require.Equal(t, m.Type, PlayerConnected)

		}
	}
}
