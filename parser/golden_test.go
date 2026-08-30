package parser

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden files")

// TestGoldenMiniNDJSON drives the full pipeline: real NDJSON fixture file ->
// readNDJSON -> Aggregate -> same encoder the CLI uses. The fixture touches
// every handled event type; the golden was generated with ./odota_cli and
// hand-verified (see AGENTS.md "Testing & CI").
func TestGoldenMiniNDJSON(t *testing.T) {
	events, err := ReadNDJSONFile("testdata/mini.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("fixture has no events")
	}
	counts := map[string]int{}
	for _, ev := range events {
		counts[ev.Type]++
	}
	for _, typ := range []string{
		"epilogue", "player_slot", "interval",
		"DOTA_COMBATLOG_DAMAGE", "DOTA_COMBATLOG_HEAL",
		"DOTA_COMBATLOG_MODIFIER_ADD", "DOTA_COMBATLOG_MODIFIER_REMOVE",
		"DOTA_COMBATLOG_PURCHASE", "DOTA_COMBATLOG_GOLD", "CHAT_MESSAGE_FIRSTBLOOD",
	} {
		if counts[typ] == 0 {
			t.Errorf("fixture missing event type %q", typ)
		}
	}

	m, err := Aggregate(events)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	wantPath := "testdata/mini.golden"
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	want := string(wantBytes)

	if *update {
		if err := os.WriteFile(wantPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", wantPath)
		return
	}

	if got != want {
		t.Errorf("output != golden.\nrun with -update only after intentional changes:\n  go test ./parser -run TestGoldenMiniNDJSON -update\n")
	}
}

func TestGoldenNoSourceKeys(t *testing.T) {
	events, err := ReadNDJSONFile("testdata/mini.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	m, err := Aggregate(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range m.Players {
		all := append([]SourceEntry(nil), p.StunSources...)
		all = append(all, p.BuffSources...)
		all = append(all, p.FearSources...)
		all = append(all, p.RootSources...)
		all = append(all, p.LeashSources...)
		all = append(all, p.TrapSources...)
		all = append(all, p.TauntSources...)
		all = append(all, p.SilenceSources...)
		all = append(all, p.BreakSources...)
		all = append(all, p.DisarmSources...)
		all = append(all, p.HealSources...)
		for _, s := range all {
			if s.Inflictor == "" || s.Category == "" {
				t.Errorf("player %s has empty source entry: %+v", p.Hero, s)
			}
		}
	}
}

func TestGoldenRadiantWinAndMetadata(t *testing.T) {
	events, err := ReadNDJSONFile("testdata/mini.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	m, err := Aggregate(events)
	if err != nil {
		t.Fatal(err)
	}
	if m.MatchID != 8926354517 || m.DurationSec != 600 || !m.RadiantWin {
		t.Errorf("metadata = %+v", m)
	}
	if len(m.Players) != 4 {
		t.Fatalf("len(players) = %d", len(m.Players))
	}
	for i, p := range m.Players {
		if p.PlayerID != i {
			t.Errorf("players[%d].PlayerID = %d", i, p.PlayerID)
		}
	}
}
