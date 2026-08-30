package parser

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ---- builders (construct RawEvents the way odota/parser emits them) ----

func iptr(v int) *int         { return &v }
func bptr(v bool) *bool       { return &v }
func fptr(v float64) *float64 { return &v }

func heroBytes(s string) []int {
	out := make([]int, len(s))
	for i := range out {
		out[i] = int(s[i])
	}
	return out
}

type pbBytes struct {
	Bytes []int `json:"bytes"`
}

type pbPlayer struct {
	SteamID    json.RawMessage `json:"steamid_"`
	HeroName   pbBytes         `json:"heroName_"`
	PlayerName pbBytes         `json:"playerName_"`
	GameTeam   int             `json:"gameTeam_"`
}

func heroPlayer(steamID int64, hero, name string, team int) pbPlayer {
	return pbPlayer{
		SteamID:    json.RawMessage(strconv.FormatInt(steamID, 10)),
		HeroName:   pbBytes{Bytes: heroBytes(hero)},
		PlayerName: pbBytes{Bytes: heroBytes(name)},
		GameTeam:   team,
	}
}

// epilogueEv builds the epilogue RawEvent; key is a protobuf-as-JSON string.
func epilogueEv(matchID int64, dur float64, winner int, players []pbPlayer) RawEvent {
	raw := map[string]any{
		"playbackTime_": dur,
		"gameInfo_": map[string]any{
			"dota_": map[string]any{
				"matchId_":    matchID,
				"gameWinner_": winner,
				"playerInfo_": players,
			},
		},
	}
	b, err := json.Marshal(raw)
	if err != nil {
		panic(err)
	}
	key, _ := json.Marshal(string(b))
	return RawEvent{Type: "epilogue", Key: key}
}

func slotEv(slot, val int) RawEvent {
	return RawEvent{
		Type:  "player_slot",
		Key:   json.RawMessage(strconv.Quote(strconv.Itoa(slot))),
		Value: json.RawMessage(strconv.Itoa(val)),
	}
}

func unitEv(t, slot int, unit string) RawEvent {
	return RawEvent{Type: "interval", Time: t, Slot: iptr(slot), Unit: unit}
}

func statEv(t, slot, gold, xp int) RawEvent {
	return RawEvent{Type: "interval", Time: t, Slot: iptr(slot), Gold: gold, Xp: xp}
}

func modEv(kind string, t int, attacker, target, inf string) RawEvent {
	return RawEvent{
		Type: kind, Time: t,
		AttackerName: attacker, TargetName: target, Inflictor: inf,
		Attackerhero: bptr(true), Targethero: bptr(true),
		Attackerillusion: bptr(false), Targetillusion: bptr(false),
	}
}

func healEv(t, val int, attacker, target, inf string) RawEvent {
	return RawEvent{
		Type: "DOTA_COMBATLOG_HEAL", Time: t, Value: json.RawMessage(strconv.Itoa(val)),
		AttackerName: attacker, TargetName: target, Inflictor: inf,
		Attackerhero: bptr(true), Targethero: bptr(true),
		Attackerillusion: bptr(false), Targetillusion: bptr(false),
	}
}

// dmgEv: nil flags keep the hero/true defaults.
func dmgEv(t, val int, attacker, target string, attackerhero, targethero, illusions *bool) RawEvent {
	e := modEv("DOTA_COMBATLOG_DAMAGE", t, attacker, target, "dota_unknown")
	e.Value = json.RawMessage(strconv.Itoa(val))
	if attackerhero != nil {
		e.Attackerhero = attackerhero
	}
	if targethero != nil {
		e.Targethero = targethero
	}
	if illusions != nil {
		e.Attackerillusion = illusions
		e.Targetillusion = illusions
	}
	return e
}

func purchaseEv(t int, target, item string) RawEvent {
	return RawEvent{Type: "DOTA_COMBATLOG_PURCHASE", Time: t, TargetName: target, Valuename: item}
}

func goldEv(t, val int, reason *int, target string) RawEvent {
	return RawEvent{
		Type: "DOTA_COMBATLOG_GOLD", Time: t, Value: json.RawMessage(strconv.Itoa(val)),
		GoldReason: reason, TargetName: target,
	}
}

func lifeEv(t, slot, state int) RawEvent {
	return RawEvent{Type: "interval", Time: t, Slot: iptr(slot), LifeState: iptr(state)}
}

func firstBloodEv(slot int) RawEvent {
	return RawEvent{
		Type:    "CHAT_MESSAGE_FIRSTBLOOD",
		Player1: json.RawMessage(strconv.Itoa(slot)),
	}
}

const (
	treant  = "npc_dota_hero_treant"
	rubick  = "npc_dota_hero_rubick"
	lich    = "npc_dota_hero_lich"
	phoenix = "npc_dota_hero_phoenix"
)

// twoPlayerBase: treant (radiant, slot0), rubick (dire, slot1).
func twoPlayerBase(t *testing.T, tail ...RawEvent) []RawEvent {
	t.Helper()
	events := []RawEvent{
		epilogueEv(8926354517, 600, 2, []pbPlayer{
			heroPlayer(76561198863696823, treant, "TA", 2),
			heroPlayer(76561199192587348, rubick, "RB", 3),
		}),
		slotEv(0, 0),
		slotEv(1, 128),
		unitEv(-89, 0, "CDOTA_Unit_Hero_Treant"),
		unitEv(-89, 1, "CDOTA_Unit_Hero_Rubick"),
		statEv(600, 0, 0, 0),
		statEv(600, 1, 0, 0),
	}
	return append(events, tail...)
}

// fourPlayerBase: adds lich (radiant, slot2), phoenix (dire, slot3).
func fourPlayerBase(t *testing.T, tail ...RawEvent) []RawEvent {
	t.Helper()
	events := []RawEvent{
		epilogueEv(8926354517, 600, 3, []pbPlayer{
			heroPlayer(76561198863696823, treant, "TA", 2),
			heroPlayer(76561199192587348, rubick, "RB", 3),
			heroPlayer(76561198196019611, lich, "LC", 2),
			heroPlayer(76561199370624370, phoenix, "PX", 3),
		}),
		slotEv(0, 0),
		slotEv(1, 128),
		slotEv(2, 2),
		slotEv(3, 130),
		unitEv(-89, 0, "CDOTA_Unit_Hero_Treant"),
		unitEv(-89, 1, "CDOTA_Unit_Hero_Rubick"),
		unitEv(-89, 2, "CDOTA_Unit_Hero_Lich"),
		unitEv(-89, 3, "CDOTA_Unit_Hero_Phoenix"),
		statEv(600, 0, 0, 0),
		statEv(600, 1, 0, 0),
		statEv(600, 2, 0, 0),
		statEv(600, 3, 0, 0),
	}
	return append(events, tail...)
}

func aggregateOrFatal(t *testing.T, events []RawEvent) *Match {
	t.Helper()
	m, err := Aggregate(events)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	return m
}

func playerOf(t *testing.T, m *Match, hero string) Player {
	t.Helper()
	for _, p := range m.Players {
		if p.Hero == hero {
			return p
		}
	}
	t.Fatalf("player %q not found; players=%v", hero, m.Players)
	return Player{}
}

// ---- scenarios ----

func TestAggregateEpilogue(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t))
	if m.MatchID != 8926354517 {
		t.Errorf("MatchID = %d", m.MatchID)
	}
	if m.DurationSec != 600 {
		t.Errorf("DurationSec = %v", m.DurationSec)
	}
	if !m.RadiantWin {
		t.Error("gameWinner_=2 should be radiant win")
	}
	if len(m.Players) != 2 {
		t.Fatalf("len(players) = %d", len(m.Players))
	}
	tA := playerOf(t, m, "treant")
	if tA.SteamID != 76561198863696823 {
		t.Errorf("treant steam_id = %d, want exact 76561198863696823", tA.SteamID)
	}
	if tA.HeroID != 83 {
		t.Errorf("treant hero_id = %d, want 83", tA.HeroID)
	}
	if tA.Team != "radiant" || tA.Name != "TA" || tA.PlayerID != 0 {
		t.Errorf("treant = %+v", tA)
	}
	if rB := playerOf(t, m, "rubick"); rB.Team != "dire" || rB.PlayerID != 1 {
		t.Errorf("rubick = %+v", rB)
	}
}

func TestAggregateDireWin(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t))
	if m.RadiantWin {
		t.Error("gameWinner_=3 should be dire win")
	}
}

func TestPlayerSlotBit128(t *testing.T) {
	// dire value 130 (not exactly 128) maps to dire via the 0x80 bit.
	m := aggregateOrFatal(t, fourPlayerBase(t))
	if px := playerOf(t, m, "phoenix"); px.Team != "dire" {
		t.Errorf("phoenix slot3 val130 team = %q, want dire", px.Team)
	}
	if lc := playerOf(t, m, "lich"); lc.Team != "radiant" {
		t.Errorf("lich slot2 val2 team = %q, want radiant", lc.Team)
	}
}

func TestIntervalLastOverwritesAndGPMXPM(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t,
		// slot0 earlier snapshot gets overwritten by t=1800
		statEv(720, 0, 100, 200),
		RawEvent{Type: "interval", Time: 1800, Slot: iptr(0), Gold: 40000, Xp: 80000,
			Kills: 12, Deaths: 3, Assists: 15, Level: 30, Lh: 250, Networth: 25000,
			Stuns: 22.5, SenPlaced: 4, CampsStacked: 2, CreepsStacked: 5, RunePickups: 1},
		statEv(720, 1, 15000, 30000),
		RawEvent{Type: "interval", Time: 1800, Slot: iptr(1), Gold: 35000, Xp: 70000,
			Kills: 9, Level: 28, Networth: 22000},
	))
	tA := playerOf(t, m, "treant")
	if tA.Kills != 12 || tA.Deaths != 3 || tA.Assists != 15 || tA.Level != 30 || tA.LastHits != 250 {
		t.Errorf("treant interval stats = %+v", tA)
	}
	// lastIntervalTime=1800s => 30 min game
	if tA.GPM != 1333 || tA.XPM != 2667 {
		t.Errorf("treant GPM/XPM = %d/%d, want 1333/2667", tA.GPM, tA.XPM)
	}
	if tA.StunDuration != 22.5 {
		t.Errorf("treant stun_duration = %v, want 22.5", tA.StunDuration)
	}
	if tA.GoldSpentWards != 200 {
		t.Errorf("treant gold_spent_wards = %d, want 200 (4 sen * 50)", tA.GoldSpentWards)
	}
	if tA.CampsStacked != 2 || tA.CreepsStacked != 5 || tA.RunePickups != 1 {
		t.Errorf("treant misc = camps:%d creeps:%d runes:%d", tA.CampsStacked, tA.CreepsStacked, tA.RunePickups)
	}
	rB := playerOf(t, m, "rubick")
	if rB.GPM != 1167 || rB.XPM != 2333 || rB.Networth != 22000 {
		t.Errorf("rubick = %d/%d/%d", rB.GPM, rB.XPM, rB.Networth)
	}
}

func TestHealingSelfAndEnemyExcluded(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t,
		healEv(2, 300, rubick, treant, "dota_unknown"),         // enemy -> excluded
		healEv(3, 400, treant, treant, "dota_unknown"),         // self -> excluded
		healEv(4, 50, "dota_fountain", treant, "dota_unknown"), // non-hero attacker -> excluded
	))
	tA := playerOf(t, m, "treant")
	if tA.Healing != 0 {
		t.Errorf("treant healing = %d, want 0", tA.Healing)
	}
}

func TestHealingSameTeamAndInstantItems(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		healEv(2, 125, treant, lich, "item_holy_locket"),     // ally instant -> Value
		healEv(3, 40, treant, lich, "dota_unknown"),          // ally regular -> Healing only
		healEv(4, 300, lich, treant, "item_greater_famango"), // ally instant, attributed to lich
		healEv(5, 100, rubick, treant, "item_holy_locket"),   // enemy -> excluded
		healEv(6, 90, treant, treant, "item_holy_locket"),    // self -> excluded
	))
	tA := playerOf(t, m, "treant")
	if tA.Healing != 125+40 {
		t.Errorf("treant healing = %d, want %d", tA.Healing, 125+40)
	}
	if tA.HealValue != 125 {
		t.Errorf("treant heal_value = %v, want 125", tA.HealValue)
	}
	if len(tA.HealSources) != 1 || tA.HealSources[0].Inflictor != "item_holy_locket" || tA.HealSources[0].Value != 125 {
		t.Errorf("treant heal_sources = %+v", tA.HealSources)
	}
	lC := playerOf(t, m, "lich")
	if lC.Healing != 300 {
		t.Errorf("lich healing = %d, want 300", lC.Healing)
	}
	if lC.HealValue != 300 {
		t.Errorf("lich heal_value = %v, want 300", lC.HealValue)
	}
	if len(lC.HealSources) != 1 || lC.HealSources[0].Inflictor != "item_greater_famango" {
		t.Errorf("lich heal_sources = %+v", lC.HealSources)
	}
}

func TestDamageAttribution(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		// hero damage
		dmgEv(1, 300, treant, rubick, nil, nil, nil),
		// self damage excluded
		dmgEv(2, 142, treant, treant, nil, nil, nil),
		// illusion target excluded
		dmgEv(3, 77, treant, rubick, nil, nil, bptr(true)),
		// summon folding: sourcename is a hero, attacker is a unit
		RawEvent{
			Type: "DOTA_COMBATLOG_DAMAGE", Time: 4, Value: json.RawMessage("60"),
			AttackerName: "npc_dota_unit_wraith_king_skeleton", Sourcename: treant,
			TargetName: rubick, Attackerhero: bptr(false), Targethero: bptr(true),
			Targetillusion: bptr(false),
		},
		// tower / rax / creep
		dmgEv(5, 500, rubick, "npc_dota_building_t1_tower_bot_x", nil, bptr(false), nil),
		dmgEv(6, 100, lich, "npc_dota_building_rax_top_melee", nil, bptr(false), nil),
		dmgEv(7, 25, phoenix, "npc_dota_creep_lane_goodguys_1", nil, bptr(false), nil),
		// damage taken
		dmgEv(8, 200, rubick, treant, nil, nil, nil),
		dmgEv(9, 150, phoenix, lich, nil, nil, nil),
	))
	tA := playerOf(t, m, "treant")
	if tA.HeroDamage != 300+60 {
		t.Errorf("treant hero_damage = %d, want 360 (incl. summon fold)", tA.HeroDamage)
	}
	if tA.DamageTaken != 200 {
		t.Errorf("treant damage_taken = %d, want 200", tA.DamageTaken)
	}
	if tA.TowerDamage != 0 {
		t.Errorf("treant tower_damage = %d, want 0", tA.TowerDamage)
	}
	rB := playerOf(t, m, "rubick")
	if rB.HeroDamage != 200 {
		t.Errorf("rubick hero_damage = %d, want 200", rB.HeroDamage)
	}
	if rB.DamageTaken != 300 {
		t.Errorf("rubick damage_taken = %d, want 300 (summon has no attackerhero)", rB.DamageTaken)
	}
	if rB.TowerDamage != 500 {
		t.Errorf("rubick tower_damage = %d, want 500", rB.TowerDamage)
	}
	lC := playerOf(t, m, "lich")
	if lC.DamageTaken != 150 || lC.TowerDamage != 100 {
		t.Errorf("lich taken/tower = %d/%d, want 150/100", lC.DamageTaken, lC.TowerDamage)
	}
	pX := playerOf(t, m, "phoenix")
	if pX.HeroDamage != 150 || pX.DamageTaken != 0 {
		t.Errorf("phoenix hero/taken = %d/%d", pX.HeroDamage, pX.DamageTaken)
	}
}

func TestSelfDamageWithSourcenameHero(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		RawEvent{
			Type: "DOTA_COMBATLOG_DAMAGE", Time: 1, Value: json.RawMessage("999"),
			AttackerName: phoenix, Sourcename: phoenix, TargetName: phoenix,
			Attackerhero: bptr(true), Targethero: bptr(true), Targetillusion: bptr(false),
		},
	))
	pX := playerOf(t, m, "phoenix")
	if pX.HeroDamage != 0 || pX.DamageTaken != 0 {
		t.Errorf("self damage should be excluded: hero=%d taken=%d", pX.HeroDamage, pX.DamageTaken)
	}
}

func TestModifierBuffAllyOnly(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, lich, "modifier_dazzle_shallow_grave"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 105, treant, lich, "modifier_dazzle_shallow_grave"),
		// enemy buff skipped
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 106, treant, rubick, "modifier_dazzle_shallow_grave"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 110, treant, rubick, "modifier_dazzle_shallow_grave"),
	))
	tA := playerOf(t, m, "treant")
	if tA.BuffDuration != 5 {
		t.Errorf("treant buff_duration = %v, want 5", tA.BuffDuration)
	}
	if len(tA.BuffSources) != 1 || tA.BuffSources[0].Category != "save" || tA.BuffSources[0].Duration != 5 {
		t.Errorf("treant buff_sources = %+v", tA.BuffSources)
	}
	if rB := playerOf(t, m, "rubick"); len(rB.BuffSources) != 0 || rB.BuffDuration != 0 {
		t.Errorf("enemy buff should not reach rubick: %+v", rB.BuffSources)
	}
}

func TestModifierSelfCastExcluded(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, treant, "modifier_dazzle_shallow_grave"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 105, treant, treant, "modifier_dazzle_shallow_grave"),
		// empty attacker falls back to target -> self -> skipped
		RawEvent{
			Type: "DOTA_COMBATLOG_MODIFIER_ADD", Time: 100, AttackerName: "", TargetName: lich,
			Inflictor: "modifier_dazzle_shallow_grave", Targethero: bptr(true),
		},
	))
	if tA := playerOf(t, m, "treant"); tA.BuffDuration != 0 {
		t.Errorf("self-cast buff recorded: %v", tA.BuffDuration)
	}
}

func TestModifierStunFromBuffAdd(t *testing.T) {
	e := modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, lich, "modifier_dazzle_shallow_grave")
	e.StunDuration = fptr(2.5)
	m := aggregateOrFatal(t, fourPlayerBase(t, e))
	tA := playerOf(t, m, "treant")
	if len(tA.StunSources) != 1 || tA.StunSources[0].Duration != 2.5 || tA.StunSources[0].Category != "stun" {
		t.Errorf("treant stun_sources = %+v", tA.StunSources)
	}
}

func TestModifierEnemyCategoriesAttributedToCaster(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		// fear: treant -> rubick (enemy), counted on treant
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 200, treant, rubick, "modifier_terrorize"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 206, treant, rubick, "modifier_terrorize"),
		// fear ally treant -> lich skipped
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 207, treant, lich, "modifier_terrorize"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 210, treant, lich, "modifier_terrorize"),
		// root: rubick -> treant
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 300, rubick, treant, "modifier_naga_siren_ensnare"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 304, rubick, treant, "modifier_naga_siren_ensnare"),
		// trap: lich -> rubick (kinetic field)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 400, lich, rubick, "modifier_disruptor_kinetic_field"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 405, lich, rubick, "modifier_disruptor_kinetic_field"),
		// taunt: phoenix -> treant (berserkers_call)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 500, phoenix, treant, "modifier_axe_berserkers_call"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 505, phoenix, treant, "modifier_axe_berserkers_call"),
		// silence: rubick -> lich (orchid)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 600, rubick, lich, "modifier_orchid_malevolence"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 602, rubick, lich, "modifier_orchid_malevolence"),
		// break: lich -> phoenix (silver edge)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 700, lich, phoenix, "modifier_item_silver_edge_debuff"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 702, lich, phoenix, "modifier_item_silver_edge_debuff"),
		// disarm: rubick -> treant (heavens halberd)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 800, rubick, treant, "modifier_heavens_halberd_debuff"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 803, rubick, treant, "modifier_heavens_halberd_debuff"),
		// HoT heal urn: treant -> lich (ally)
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 900, treant, lich, "modifier_item_urn_heal"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 912, treant, lich, "modifier_item_urn_heal"),
	))

	tA := playerOf(t, m, "treant")
	if tA.FearDuration != 6 {
		t.Errorf("treant fear_duration = %v, want 6", tA.FearDuration)
	}
	if len(tA.FearSources) != 1 || tA.FearSources[0].Category != "fear" {
		t.Errorf("treant fear_sources = %+v", tA.FearSources)
	}
	if tA.RootsDuration != 0 || tA.TauntDuration != 0 || tA.DisarmDuration != 0 {
		t.Errorf("treant should have no root/taunt/disarm (all enemy-applied to it): %+v", tA)
	}
	if tA.HealDuration != 12 {
		t.Errorf("treant heal_duration = %v, want 12 (urn HoT)", tA.HealDuration)
	}

	rB := playerOf(t, m, "rubick")
	if rB.RootsDuration != 4 {
		t.Errorf("rubick roots_duration = %v, want 4", rB.RootsDuration)
	}
	if len(rB.RootSources) != 1 || rB.RootSources[0].Category != "root" {
		t.Errorf("rubick root_sources = %+v", rB.RootSources)
	}
	if rB.SilenceDuration != 2 {
		t.Errorf("rubick silence_duration = %v, want 2", rB.SilenceDuration)
	}
	if rB.DisarmDuration != 3 {
		t.Errorf("rubick disarm_duration = %v, want 3", rB.DisarmDuration)
	}
	if rB.TrapDuration != 0 {
		t.Errorf("rubick is target of kinetic field; trap attributed to lich, got %v", rB.TrapDuration)
	}

	lC := playerOf(t, m, "lich")
	if lC.TrapDuration != 5 {
		t.Errorf("lich trap_duration = %v, want 5", lC.TrapDuration)
	}
	if lC.BreakDuration != 2 {
		t.Errorf("lich break_duration = %v, want 2 (lich applied silver edge)", lC.BreakDuration)
	}

	pX := playerOf(t, m, "phoenix")
	if pX.TauntDuration != 5 {
		t.Errorf("phoenix taunt_duration = %v, want 5 (phoenix applied berserkers_call)", pX.TauntDuration)
	}
	if pX.BreakDuration != 0 {
		t.Errorf("phoenix is target of break; break attributed to lich, got %v", pX.BreakDuration)
	}
}

func TestModifierLIFOOrdering(t *testing.T) {
	// two overlapping fears; LIFO pops newest first, both sum to the same source
	m := aggregateOrFatal(t, fourPlayerBase(t,
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, rubick, "modifier_terrorize"),
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 110, treant, rubick, "modifier_terrorize"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 115, treant, rubick, "modifier_terrorize"), // pops t=110
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 130, treant, rubick, "modifier_terrorize"), // pops t=100
	))
	tA := playerOf(t, m, "treant")
	if tA.FearDuration != 35 {
		t.Errorf("LIFO fear_duration = %v, want 35 ((130-110)+(115-100))", tA.FearDuration)
	}
}

func TestModifierRemoveWithoutAddIgnored(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 500, treant, rubick, "modifier_terrorize"),
	))
	if tA := playerOf(t, m, "treant"); tA.FearDuration != 0 {
		t.Errorf("remove without add produced fear: %v", tA.FearDuration)
	}
}

func TestModifierNonHeroAndUnknownTargetsIgnored(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		// fountain buff on hero: attacker not in BuffCategories -> ignored
		RawEvent{
			Type: "DOTA_COMBATLOG_MODIFIER_ADD", Time: 1, AttackerName: "dota_fountain",
			TargetName: treant, Inflictor: "modifier_fountain_aura_buff", Targethero: bptr(true),
		},
		RawEvent{
			Type: "DOTA_COMBATLOG_MODIFIER_REMOVE", Time: 2, AttackerName: "dota_fountain",
			TargetName: treant, Inflictor: "modifier_fountain_aura_buff", Targethero: bptr(true),
		},
		// non-hero target ignored
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 3, treant, "npc_dota_creep_lane_badguys_1", "modifier_terrorize"),
	))
	tA := playerOf(t, m, "treant")
	if tA.BuffDuration != 0 || tA.FearDuration != 0 {
		t.Errorf("ignored modifiers recorded: buff=%v fear=%v", tA.BuffDuration, tA.FearDuration)
	}
}

func TestPurchaseSmokeDust(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t,
		purchaseEv(-89, treant, "item_smoke_of_deceit"),
		purchaseEv(-85, treant, "item_smoke_of_deceit"),
		purchaseEv(-80, rubick, "item_dust"),
		purchaseEv(-79, rubick, "item_ward_dispenser"), // not tracked
		purchaseEv(-78, rubick, ""),                    // no valuename
	))
	tA := playerOf(t, m, "treant")
	if tA.GoldSpentSmoke != 100 || tA.GoldSpentDust != 0 {
		t.Errorf("treant smoke/dust = %d/%d, want 100/0", tA.GoldSpentSmoke, tA.GoldSpentDust)
	}
	if rB := playerOf(t, m, "rubick"); rB.GoldSpentDust != 80 {
		t.Errorf("rubick dust = %d, want 80", rB.GoldSpentDust)
	}
}

func TestPurchaseShortVariantName(t *testing.T) {
	// pre-game purchases log the hero under the underscore-less variant; heroKey folds them
	m := aggregateOrFatal(t, twoPlayerBase(t,
		purchaseEv(-89, "npc_dota_hero_witchdoctor", "item_smoke_of_deceit"),
		purchaseEv(-88, rubick, "item_dust"),
	))
	// witchdoctor is not a player; ignored cleanly
	if rB := playerOf(t, m, "rubick"); rB.GoldSpentDust != 80 {
		t.Errorf("rubick dust = %d", rB.GoldSpentDust)
	}
}

func TestDeathGoldLost(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t,
		goldEv(1, -200, iptr(1), treant),
		goldEv(2, -80, iptr(1), treant),
		goldEv(3, -50, iptr(2), treant),                      // not a death loss (reason 2)
		goldEv(4, 30, nil, treant),                           // no reason
		goldEv(5, -25, iptr(1), "npc_dota_hero_witchdoctor"), // non-player, ignored
	))
	if tA := playerOf(t, m, "treant"); tA.GoldLost != 280 {
		t.Errorf("treant gold_lost = %d, want 280", tA.GoldLost)
	}
}

func TestDeathGoldLostPositiveValue(t *testing.T) {
	// death loss logs a negative value; abs() recovers it regardless of sign
	m := aggregateOrFatal(t, twoPlayerBase(t, goldEv(1, 300, iptr(1), treant)))
	if tA := playerOf(t, m, "treant"); tA.GoldLost != 300 {
		t.Errorf("treant gold_lost = %d, want 300", tA.GoldLost)
	}
}

func TestTimeDeadLifeState(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t,
		lifeEv(500, 0, 0), // alive
		lifeEv(501, 0, 1), // dies
		lifeEv(502, 0, 1),
		lifeEv(503, 0, 1),
		lifeEv(504, 0, 0), // respawns
		lifeEv(505, 0, 0),
		// buyback after a second death: dead window cut short
		lifeEv(510, 0, 1), // dies
		lifeEv(511, 0, 0), // buyback -> alive next tick
	))
	if tA := playerOf(t, m, "treant"); tA.TimeDead != 4 {
		t.Errorf("treant time_dead = %v, want 4 (3s window + 1s buyback)", tA.TimeDead)
	}
}

func TestTimeDeadStillDeadAtEnd(t *testing.T) {
	// hero dead from t=700 until the last interval (t=900); life_state after the
	// final snapshot (which carries the last stats) keeps the death open to 900
	m := aggregateOrFatal(t, twoPlayerBase(t,
		lifeEv(600, 0, 0),
		lifeEv(700, 0, 1),
		lifeEv(900, 0, 1), // last interval, still dead
	))
	if tA := playerOf(t, m, "treant"); tA.TimeDead != 200 {
		t.Errorf("treant time_dead = %v, want 200 (700..900)", tA.TimeDead)
	}
}

func TestTimeDeadNoLifeState(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t))
	if tA := playerOf(t, m, "treant"); tA.TimeDead != 0 {
		t.Errorf("no life_state should yield time_dead 0, got %v", tA.TimeDead)
	}
}

func TestFirstBlood(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t, firstBloodEv(1)))
	if !playerOf(t, m, "rubick").FirstBlood {
		t.Error("rubick (slot 1) should have first_blood")
	}
	if playerOf(t, m, "treant").FirstBlood {
		t.Error("treant should not have first_blood")
	}
}

func TestEpilogueErrorPropagates(t *testing.T) {
	bad := RawEvent{Type: "epilogue", Key: json.RawMessage(`{not json`)}
	if _, err := Aggregate([]RawEvent{bad}); err == nil {
		t.Error("malformed epilogue should error")
	}
	noGameInfo := RawEvent{Type: "epilogue", Key: json.RawMessage(`"{\"playbackTime_\":1}"`)}
	if _, err := Aggregate([]RawEvent{noGameInfo}); err == nil {
		t.Error("epilogue without gameInfo_ should error")
	}
}

func TestParseEpilogueDirect(t *testing.T) {
	ev := epilogueEv(42, 100.5, 3, []pbPlayer{
		heroPlayer(76561198863696823, treant, "TA", 2),
		heroPlayer(76561199192587348, rubick, "RB", 3),
	})
	var keyStr string
	if err := json.Unmarshal(ev.Key, &keyStr); err != nil {
		t.Fatal(err)
	}
	r, err := parseEpilogue(keyStr)
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchID != 42 || r.DurationSec != 100.5 || r.RadiantWin {
		t.Errorf("epilogue result = %+v", r)
	}
	if r.Players[0].SteamID != 76561198863696823 || r.Players[0].HeroID != 83 {
		t.Errorf("player0 = %+v", r.Players[0])
	}
	if r.Players[1].Team != "dire" {
		t.Errorf("player1 team = %q", r.Players[1].Team)
	}
	if _, err := parseEpilogue("not json"); err == nil {
		t.Error("parseEpilogue(garbage) should error")
	}
}

func TestOutputSourceSorting(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t,
		// equal-duration tie broken by inflictor ascending
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, lich, "modifier_item_aeon_disk"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 105, treant, lich, "modifier_item_aeon_disk"),
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 106, treant, lich, "modifier_ogre_magi_bloodlust"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 109, treant, lich, "modifier_ogre_magi_bloodlust"),
		// shorter buff sorts after the equal pair
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 120, treant, lich, "modifier_dark_seer_surge"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 121, treant, lich, "modifier_dark_seer_surge"),
	))
	tA := playerOf(t, m, "treant")
	wantOrder := []string{"item_aeon_disk", "ogre_magi_bloodlust", "dark_seer_surge"}
	if len(tA.BuffSources) != 3 {
		t.Fatalf("treant buff_sources = %+v", tA.BuffSources)
	}
	for i, want := range wantOrder {
		if tA.BuffSources[i].Inflictor != want {
			t.Errorf("buff_sources[%d] = %q, want %q (det: %v)", i, tA.BuffSources[i].Inflictor, want, tA.BuffSources)
		}
	}
}

func TestOutputJSONOmitEmpty(t *testing.T) {
	m := aggregateOrFatal(t, twoPlayerBase(t))
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, key := range []string{
		"stun_sources", "buff_sources", "fear_sources", "root_sources", "leash_sources",
		"trap_sources", "taunt_sources", "silence_sources", "break_sources",
		"disarm_sources", "heal_sources",
	} {
		if strings.Contains(s, `"`+key+`"`) {
			t.Errorf("empty %s should be omitted", key)
		}
	}
}

func TestAggregateDeterministic(t *testing.T) {
	events := fourPlayerBase(t,
		healEv(2, 125, treant, lich, "item_holy_locket"),
		dmgEv(1, 300, treant, rubick, nil, nil, nil),
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 100, treant, lich, "modifier_dazzle_shallow_grave"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 105, treant, lich, "modifier_dazzle_shallow_grave"),
		modEv("DOTA_COMBATLOG_MODIFIER_ADD", 200, treant, rubick, "modifier_terrorize"),
		modEv("DOTA_COMBATLOG_MODIFIER_REMOVE", 206, treant, rubick, "modifier_terrorize"),
	)
	m1 := aggregateOrFatal(t, events)
	m2 := aggregateOrFatal(t, events)
	b1, _ := json.Marshal(m1)
	b2, _ := json.Marshal(m2)
	if string(b1) != string(b2) {
		t.Error("Aggregate output not deterministic across runs")
	}
}

func TestSortTiebreakStability(t *testing.T) {
	entries := []SourceEntry{
		{Inflictor: "zzz", Duration: 5},
		{Inflictor: "aaa", Duration: 5},
		{Inflictor: "mmm", Duration: 9},
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Duration != entries[j].Duration {
			return entries[i].Duration > entries[j].Duration
		}
		return entries[i].Inflictor < entries[j].Inflictor
	})
	want := []string{"mmm", "aaa", "zzz"}
	for i, w := range want {
		if entries[i].Inflictor != w {
			t.Errorf("sorted[%d] = %q, want %q", i, entries[i].Inflictor, w)
		}
	}
}

func TestHealSourcesSortValueThenDuration(t *testing.T) {
	entries := []SourceEntry{
		{Inflictor: "a", Value: 10},
		{Inflictor: "b", Value: 50},
		{Inflictor: "c", Value: 50, Duration: 2},
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Duration != entries[j].Duration {
			return entries[i].Duration > entries[j].Duration
		}
		if entries[i].Value != entries[j].Value {
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Inflictor < entries[j].Inflictor
	})
	want := []string{"c", "b", "a"}
	if len(entries) != 3 {
		t.Fatalf("entries = %+v", entries)
	}
	for i, w := range want {
		if entries[i].Inflictor != w {
			t.Errorf("heal-sorted[%d] = %q, want %q", i, entries[i].Inflictor, w)
		}
	}
}

func TestOutputPlayersSortedBySlot(t *testing.T) {
	m := aggregateOrFatal(t, fourPlayerBase(t))
	for i, p := range m.Players {
		if p.PlayerID != i {
			t.Errorf("players[%d].PlayerID = %d", i, p.PlayerID)
		}
	}
}
