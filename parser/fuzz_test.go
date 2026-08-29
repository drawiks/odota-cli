package parser

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func FuzzParseInt64(f *testing.F) {
	for _, seed := range []string{
		"76561198863696823", `"76561198863696823"`, "59.5", "-7",
		`{"high":17825793,"low":903431095}`, "abc", "", "null",
		"9223372036854775807", `"-9223372036854775808"`, "1e300", "[]",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		parseInt64(json.RawMessage(raw)) // must never panic
	})
}

func FuzzParseRawFloat(f *testing.F) {
	for _, seed := range []string{"42", "1.5", "1e300", "-0.25", "null", `"5"`, "{}", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		parseRawFloat(json.RawMessage(raw))
	})
}

func FuzzClassifiers(f *testing.F) {
	for _, seed := range []string{
		"modifier_break", "modifier_silver_edge_debuff", "modifier_terrorize",
		"modifier_naga_siren_ensnare", "modifier_puck_coiled", "modifier_disruptor_kinetic_field",
		"modifier_axe_berserkers_call", "modifier_global_silence", "modifier_heavens_halberd_debuff",
		"modifier_item_urn_heal", "modifier_silencer_last_word_disarm", "",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		canonicalName(cleanInflictor(s))
		classifyBreak(s)
		classifyFear(s)
		classifyRoot(s)
		classifyLeash(s)
		classifyTrap(s)
		classifyTaunt(s)
		classifySilence(s)
		classifyDisarm(s)
		_ = classifyHeal(s)
	})
}

func FuzzHeroNames(f *testing.F) {
	for _, seed := range []string{
		"CDOTA_Unit_Hero_SkeletonKing", "CDOTA_Unit_Hero_QueenOfPain",
		"npc_dota_hero_queen_of_pain", "npc_dota_hero_queenofpain", "CDOTA_Unit_Hero_", "",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		unitToHeroName(s)
		_ = heroKey(s)
	})
}

func FuzzReadNDJSON(f *testing.F) {
	for _, seed := range []string{
		"", "{}", "null", "{\"type\":\"epilogue\"}\n",
		"{\"type\":\"interval\",\"slot\":2}\n{\"type\":\"dmg\"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		events, err := readNDJSON(bytes.NewReader([]byte(data)))
		if err != nil {
			return
		}
		_ = events
	})
}

func FuzzAggregate(f *testing.F) {
	base, err := json.Marshal(epilogueEv(8926354517, 600, 2, []pbPlayer{
		heroPlayer(76561198863696823, treant, "TA", 2),
		heroPlayer(76561199192587348, rubick, "RB", 3),
	}))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(base))
	for _, seed := range []string{
		string(base),
		string(base) + "\n{\"type\":\"DOTA_COMBATLOG_MODIFIER_ADD\",\"time\":1,\"attackername\":\"npc_dota_hero_rubick\",\"targetname\":\"npc_dota_hero_treant\",\"inflictor\":\"modifier_terrorize\"}",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		events, err := readNDJSON(strings.NewReader(data))
		if err != nil {
			return
		}
		Aggregate(events) // must never panic
	})
}
