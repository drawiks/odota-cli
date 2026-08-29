package parser

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func marshalEvents(t *testing.T, evs []RawEvent) []byte {
	t.Helper()
	var buf bytes.Buffer
	for _, ev := range evs {
		b, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func TestReadNDJSONPreservesOrder(t *testing.T) {
	in := marshalEvents(t, []RawEvent{
		{Type: "epilogue", Time: 1},
		{Type: "interval", Time: 2, Slot: iptr(3)},
		{Time: 3, AttackerName: "npc_dota_hero_treant", Attackerhero: bptr(true)},
	})
	events, err := readNDJSON(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].Type != "epilogue" || events[1].Type != "interval" || events[2].AttackerName != "npc_dota_hero_treant" {
		t.Errorf("order/fields wrong: %+v", events)
	}
	if events[1].Slot == nil || *events[1].Slot != 3 {
		t.Errorf("slot not parsed: %v", events[1].Slot)
	}
	if events[2].Attackerhero == nil || !*events[2].Attackerhero {
		t.Errorf("attackerhero not parsed: %v", events[2].Attackerhero)
	}
}

func TestReadNDJSONEmptyAndWhitespace(t *testing.T) {
	for _, in := range []string{"", "\n", " \n\n  \n"} {
		events, err := readNDJSON(strings.NewReader(in))
		if err != nil {
			t.Fatalf("readNDJSON(%q): %v", in, err)
		}
		if len(events) != 0 {
			t.Errorf("readNDJSON(%q) = %d events, want 0", in, len(events))
		}
	}
}

func TestReadNDJSONRootPrimitives(t *testing.T) {
	// a bare JSON object parses to an empty event; which Aggregate then ignores
	for _, in := range []string{"{}", "null", "{}\n{}"} {
		events, err := readNDJSON(bytes.NewReader([]byte(in)))
		if err != nil {
			t.Errorf("readNDJSON(%q) error: %v", in, err)
		}
		if len(events) != 1 && in != "{}\n{}" && len(events) != 2 {
			t.Errorf("readNDJSON(%q) = %d events", in, len(events))
		}
	}
}

func TestReadNDJSONMalformedLine(t *testing.T) {
	for _, in := range []string{"{\"type\":\n", "not json\n", "{\"type\":\"x\"} garbage\n"} {
		_, err := readNDJSON(strings.NewReader(in))
		if err == nil {
			t.Errorf("readNDJSON(%q) should error", in)
			continue
		}
		if !strings.Contains(err.Error(), "parse line") {
			t.Errorf("readNDJSON(%q) err = %v, want 'parse line'", in, err)
		}
	}
}

func TestReadNDJSONScannerBuffer(t *testing.T) {
	// a line larger than the 64KiB default scanner buffer must not error
	big := strings.Repeat("x", 256*1024)
	ev := RawEvent{Type: "epilogue", Key: json.RawMessage(`"` + big + `"`)}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 256*1024 {
		t.Fatalf("test line too small: %d", len(b))
	}
	events, err := readNDJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
}

func TestReadNDJSONFileMissing(t *testing.T) {
	if _, err := ReadNDJSONFile(filepath.Join(t.TempDir(), "nope.ndjson")); err == nil {
		t.Error("missing file should error")
	}
}

func TestFetchFromParserOK(t *testing.T) {
	in := marshalEvents(t, []RawEvent{
		{Type: "epilogue", Time: 5},
		{Type: "interval", Time: 6, Slot: iptr(0)},
	})
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.Write(in)
	}))
	defer srv.Close()

	events, err := FetchFromParser([]byte("fake dem"), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if gotCT != "application/octet-stream" {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if len(events) != 2 || events[0].Type != "epilogue" {
		t.Errorf("events = %+v", events)
	}
}

func TestFetchFromParserHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := FetchFromParser([]byte("fake dem"), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "parser returned 500") {
		t.Errorf("err = %v, want 'parser returned 500'", err)
	}
}

func TestFetchFromParserMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"type\": broken\n"))
	}))
	defer srv.Close()

	_, err := FetchFromParser([]byte("fake dem"), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "parse line") {
		t.Errorf("err = %v, want 'parse line'", err)
	}
}

func TestParseDem(t *testing.T) {
	in := marshalEvents(t, []RawEvent{
		{Type: "epilogue", Time: 30, Key: epilogueEv(7, 30, 2, []pbPlayer{heroPlayer(76561198000000001, treant, "TA", 2), heroPlayer(76561198000000002, rubick, "RB", 3)}).Key},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(in)
	}))
	defer srv.Close()

	m, err := ParseDem(bytes.NewReader([]byte("fake dem")), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if m.MatchID != 7 || !m.RadiantWin || len(m.Players) != 2 {
		t.Errorf("match = %+v", m)
	}
}

func TestFetchFromParserConnectionError(t *testing.T) {
	_, err := FetchFromParser([]byte("fake dem"), "http://127.0.0.1:1")
	if err == nil {
		t.Error("connection refused should error")
	}
}

func TestParseRawFloatJSONEdges(t *testing.T) {
	raw := json.RawMessage(`1e300`)
	if got := parseRawFloat(raw); got != 1e300 {
		t.Errorf("parseRawFloat(1e300) = %v", got)
	}
}
