package parser

import (
	"encoding/json"
	"testing"
)

func TestUnitToHeroName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"CDOTA_Unit_Hero_SkeletonKing", "npc_dota_hero_skeleton_king"},
		{"CDOTA_Unit_Hero_QueenOfPain", "npc_dota_hero_queen_of_pain"},
		{"CDOTA_Unit_Hero_ShadowShaman", "npc_dota_hero_shadow_shaman"},
		{"CDOTA_Unit_Hero_AbyssalUnderlord", "npc_dota_hero_abyssal_underlord"},
		{"CDOTA_Unit_Hero_Treant", "npc_dota_hero_treant"},
		{"CDOTA_Unit_Hero_Treant_mediocre", "npc_dota_hero_treant_mediocre"},
		{"CDOTA_Unit_Hero_Windrunner", "npc_dota_hero_windrunner"},
		{"CDOTA_Unit_Hero_Mirana", "npc_dota_hero_mirana"},
		{"CDOTA_Unit_Hero_", "npc_dota_hero_"},
		{"npc_dota_hero_phoenix", "npc_dota_hero_phoenix"},
		{"dota_creep_lane_badguys_1", "dota_creep_lane_badguys_1"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := unitToHeroName(tt.in); got != tt.want {
			t.Errorf("unitToHeroName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int64
	}{
		{"17-digit SteamID raw digits", `76561198863696823`, 76561198863696823},
		{"17-digit SteamID quoted string exact via ParseInt", `"76561198863696823"`, 76561198863696823},
		{"protobuf high/low pair", `{"high":17825793,"low":903431095}`, 76561198863696823},
		{"zero protobuf pair", `{"high":0,"low":0}`, 0},
		{"plain float", `123.45`, 123},
		{"quoted float", `"123.45"`, 123},
		{"quoted digits via fscanf", `"5"`, 5},
		{"negative", `-7`, -7},
		{"large int64 max", `9223372036854775807`, 9223372036854775807},
		{"int64 min", `-9223372036854775808`, -9223372036854775808},
		{"plain small int", `42`, 42},
		{"empty raw", ``, 0},
		{"whitespace", `  `, 0},
		{"garbage", `"abc"`, 0},
		{"null", `null`, 0},
		{"empty object", `{}`, 0},
		{"numbers in object only low", `{"low":7}`, 7},
		{"boolean", `true`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInt64(json.RawMessage(tt.raw)); got != tt.want {
				t.Errorf("parseInt64(%s) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseRawFloat(t *testing.T) {
	tests := []struct {
		raw  string
		want float64
	}{
		{"42", 42},
		{"1.5", 1.5},
		{"0", 0},
		{"-3.25", -3.25},
		{"null", 0},
		{`"5"`, 0},
		{`{"a":1}`, 0},
		{"", 0},
		{"garbage", 0},
	}
	for _, tt := range tests {
		if got := parseRawFloat(json.RawMessage(tt.raw)); got != tt.want {
			t.Errorf("parseRawFloat(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestIntsToTrimmedString(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want string
	}{
		{"ascii", []int{116, 114, 101, 97, 110, 116}, "treant"},
		{"trailing nulls trimmed", []int{116, 97, 0, 0}, "ta"},
		{"leading zero kept", []int{116, 114, 101, 97, 110, 116, 0, 0}, "treant"},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intsToTrimmedString(tt.in); got != tt.want {
				t.Errorf("intsToTrimmedString(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsIllusion(t *testing.T) {
	f := false
	tr := true
	if isIllusion(nil) {
		t.Error("isIllusion(nil) = true, want false")
	}
	if isIllusion(&f) {
		t.Error("isIllusion(&false) = true, want false")
	}
	if !isIllusion(&tr) {
		t.Error("isIllusion(&true) = false, want true")
	}
}

func TestIsBuilding(t *testing.T) {
	buildings := []string{
		"npc_dota_building_t1_tower_bot_x",
		"tower",
		"npc_dota_building_rax_top_melee",
		"fortress",
		"building",
		"shrine",
		"effigy",
		"npc_dota_building_bot_effigy",
	}
	for _, b := range buildings {
		if !isBuilding(b) {
			t.Errorf("isBuilding(%q) = false, want true", b)
		}
	}
	nonBuildings := []string{
		"npc_dota_hero_antimage",
		"npc_dota_creep_lane_goodguys_1",
		"npc_dota_ward_base_observer",
		"",
	}
	for _, b := range nonBuildings {
		if isBuilding(b) {
			t.Errorf("isBuilding(%q) = true, want false", b)
		}
	}
}

func TestHeroMap(t *testing.T) {
	tests := []struct {
		hero string
		id   int
	}{
		{"npc_dota_hero_antimage", 1},
		{"npc_dota_hero_nevermore", 11},
		{"npc_dota_hero_zuus", 22},
		{"npc_dota_hero_skeleton_king", 42},
		{"npc_dota_hero_furion", 53},
		{"npc_dota_hero_disruptor", 87},
		{"npc_dota_hero_phoenix", 110},
		{"npc_dota_hero_kez", 145},
		{"npc_dota_hero_largo", 155},
		{"npc_dota_hero_primal_beast", 137},
		{"npc_dota_hero_muerta", 138},
		{"npc_dota_hero_dark_willow", 119},
	}
	for _, tt := range tests {
		if got := HeroMap[tt.hero]; got != tt.id {
			t.Errorf("HeroMap[%q] = %d, want %d", tt.hero, got, tt.id)
		}
	}
	if HeroMap["npc_dota_hero_does_not_exist"] != 0 {
		t.Error("HeroMap should return 0 for unknown hero")
	}
	if len(HeroMap) < 90 {
		t.Errorf("HeroMap suspiciously small: %d entries", len(HeroMap))
	}
}

func TestHeroKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"npc_dota_hero_queen_of_pain", "npcdotaheroqueenofpain"},
		{"npc_dota_hero_queenofpain", "npcdotaheroqueenofpain"},
		{"witchdoctor", "witchdoctor"},
		{"a_b_c", "abc"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := heroKey(tt.in); got != tt.want {
			t.Errorf("heroKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanInflictor(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"modifier_dazzle_shallow_grave", "dazzle_shallow_grave"},
		{"dazzle_shallow_grave", "dazzle_shallow_grave"},
		{"modificador_x", "modificador_x"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := cleanInflictor(tt.in); got != tt.want {
			t.Errorf("cleanInflictor(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsAllyByTeam(t *testing.T) {
	// production stores heroTeam keyed by heroKey-normalized names; the lookup
	// normalizes queries internally, so raw underscores variants still match.
	team := map[string]string{
		heroKey("npc_dota_hero_treant"):      "radiant",
		heroKey("npc_dota_hero_windrunner"):  "radiant",
		heroKey("npc_dota_hero_rubick"):      "dire",
		heroKey("npc_dota_hero_queenofpain"): "dire",
	}
	if !isAllyByTeam(team, "npc_dota_hero_treant", "npc_dota_hero_windrunner") {
		t.Error("two radiant heroes should be allies")
	}
	if !isAllyByTeam(team, "npc_dota_hero_queen_of_pain", "npc_dota_hero_rubick") {
		t.Error("heroKey normalization should fold queen_of_pain to queenofpain")
	}
	if isAllyByTeam(team, "npc_dota_hero_treant", "npc_dota_hero_rubick") {
		t.Error("cross-team heroes must not be allies")
	}
	if isAllyByTeam(team, "npc_dota_hero_treant", "npc_dota_hero_unknown") {
		t.Error("unknown team must not be ally")
	}
	if isAllyByTeam(team, "npc_dota_hero_unknown", "npc_dota_hero_treant") {
		t.Error("unknown attacker must not be ally")
	}
	if isAllyByTeam(map[string]string{}, "npc_dota_hero_treant", "npc_dota_hero_treant") {
		t.Error("empty team map must never be ally")
	}
	if !isAllyByTeam(team, "npc_dota_hero_treant", "npc_dota_hero_treant") {
		t.Error("same hero same team should be ally")
	}
}
