package parser

import "testing"

// goldPosEv builds an in-window interval carrying both parser coords and gold,
// so it both samples the lane and sets roleGold (window farm signal).
func goldPosEv(t, slot int, x, y float64, gold int) RawEvent {
	e := posEv(t, slot, x, y, 0)
	e.Gold = gold
	return e
}

func TestClassifyZone(t *testing.T) {
	cases := []struct {
		x, y  float64
		want  roleZone
		label string
	}{
		{16000, 16000, zoneMid, "mid diagonal center"},
		{10000, 20000, zoneOff, "off top-left"},
		{21000, 10000, zoneSafe, "safe bottom-right"},
		{30000, 30000, zoneOther, "far corner nothing"},
	}
	for _, c := range cases {
		got := classifyZone(c.x, c.y)
		if got != c.want {
			t.Errorf("classifyZone(%v,%v)=%v want %v (%s)", c.x, c.y, got, c.want, c.label)
		}
	}
}

func TestRoleStatsDominant(t *testing.T) {
	r := &roleStats{}
	r.sample(21000, 10000)
	r.sample(21000, 10000)
	r.sample(21000, 10000)
	r.sample(16000, 16000)
	if z, ok := r.dominant(); !ok || z != zoneSafe {
		t.Fatalf("dominant 3/4 safe = (%v, %v), want safe,true", z, ok)
	}
	// below the 0.45 threshold -> no dominant zone (roamer)
	spread := &roleStats{}
	for i := 0; i < 3; i++ {
		spread.sample(16000, 16000)
		spread.sample(21000, 10000)
		spread.sample(10000, 20000)
		spread.sample(30000, 30000)
	}
	if _, ok := spread.dominant(); ok {
		t.Fatal("spread 3/12 per zone should not dominate")
	}
}

func TestLaneOf(t *testing.T) {
	if laneOf(zoneMid, "radiant") != laneMid || laneOf(zoneMid, "dire") != laneMid {
		t.Error("mid stays mid for both teams")
	}
	if laneOf(zoneSafe, "radiant") != laneSafe || laneOf(zoneOff, "radiant") != laneOff {
		t.Error("radiant zone mapping wrong")
	}
	// dire: inverted
	if laneOf(zoneSafe, "dire") != laneOff || laneOf(zoneOff, "dire") != laneSafe {
		t.Error("dire zone mapping wrong")
	}
	if laneOf(zoneOther, "radiant") != laneUnknown {
		t.Error("zoneOther unknown")
	}
}

func TestLaneName(t *testing.T) {
	want := map[int]string{1: "safe", 2: "mid", 3: "off", 4: "supp", 5: "hard_supp"}
	for pos, lane := range want {
		if laneName(pos) != lane {
			t.Errorf("laneName(%d)=%q want %q", pos, laneName(pos), lane)
		}
	}
	if laneName(0) != "" {
		t.Error("laneName(0) should be empty")
	}
}

// roleSample player with a stats object built from zone samples.
func roleSample(team string, zones []roleZone) (*Player, *roleStats) {
	p := &Player{Team: team}
	rs := &roleStats{}
	for _, z := range zones {
		rs.zones[z]++
		rs.total++
	}
	return p, rs
}

func TestAssignRolesFull212(t *testing.T) {
	treant, rsTreant := roleSample("radiant", []roleZone{zoneSafe})
	lich, rsLich := roleSample("radiant", []roleZone{zoneSafe})
	storm, rsStorm := roleSample("radiant", []roleZone{zoneMid})
	mars, rsMars := roleSample("radiant", []roleZone{zoneOff})
	es, rsEs := roleSample("radiant", []roleZone{zoneOff})
	// dire (inverted zones): off-zone = safe lane, safe-zone = off lane
	rubick, rsRubick := roleSample("dire", []roleZone{zoneOff})
	phoenix, rsPhoenix := roleSample("dire", []roleZone{zoneOff})
	puck, rsPuck := roleSample("dire", []roleZone{zoneMid})
	centaur, rsCentaur := roleSample("dire", []roleZone{zoneSafe})
	mirana, rsMirana := roleSample("dire", []roleZone{zoneSafe})

	in := []roleInput{
		{treant, rsTreant, 20000, 40000},
		{lich, rsLich, 9000, 28000},
		{storm, rsStorm, 10000, 30000},
		{mars, rsMars, 15000, 30000},
		{es, rsEs, 6000, 20000},
		{rubick, rsRubick, 18000, 35000},
		{phoenix, rsPhoenix, 8000, 21000},
		{puck, rsPuck, 9000, 26000},
		{centaur, rsCentaur, 12000, 25000},
		{mirana, rsMirana, 4000, 12000},
	}
	assignRoles(in)

	want := map[*Player]struct {
		pos  int
		lane string
	}{
		treant: {1, "safe"}, lich: {5, "hard_supp"}, storm: {2, "mid"},
		mars: {3, "off"}, es: {4, "supp"},
		rubick: {1, "safe"}, phoenix: {5, "hard_supp"}, puck: {2, "mid"},
		centaur: {3, "off"}, mirana: {4, "supp"},
	}
	for p, w := range want {
		if p.Position != w.pos || p.Lane != w.lane {
			t.Errorf("team=%s got pos=%d lane=%q want pos=%d lane=%q",
				p.Team, p.Position, p.Lane, w.pos, w.lane)
		}
	}
}

func TestAssignRolesRoamerAndOverfull(t *testing.T) {
	a, ra := roleSample("radiant", []roleZone{zoneSafe})
	b, rb := roleSample("radiant", []roleZone{zoneSafe})
	c, rc := roleSample("radiant", []roleZone{zoneSafe})
	roam, rr := roleSample("radiant", []roleZone{zoneOther, zoneSafe, zoneOff})
	mid, rm := roleSample("radiant", []roleZone{zoneMid})

	in := []roleInput{
		{a, ra, 100, 500},
		{b, rb, 100, 300},
		{c, rc, 100, 200},
		{roam, rr, 100, 400},
		{mid, rm, 100, 250},
	}
	assignRoles(in)

	// safe pairs by farm: a=carry(1), b=hard_supp(5); overfull c falls to unk.
	// Free support-first slots (offSupp=4, offCore=3) filled by farm order:
	// roamer(400)>c(200), so roamer gets offSupp(4), c gets offCore(3).
	if a.Position != 1 || b.Position != 5 {
		t.Errorf("safe pair: a=%d b=%d want 1,5", a.Position, b.Position)
	}
	if mid.Position != 2 {
		t.Errorf("mid pos=%d want 2", mid.Position)
	}
	if roam.Position != 4 || c.Position != 3 {
		t.Errorf("free fill: roam pos=%d want 4, c pos=%d want 3", roam.Position, c.Position)
	}
}

func TestAssignRolesSkipsNoSamples(t *testing.T) {
	noSamples := &Player{Team: "radiant"}
	withSamples, rs := roleSample("radiant", []roleZone{zoneMid})
	in := []roleInput{
		{noSamples, nil, 0, 0},
		{withSamples, rs, 100, 100},
	}
	assignRoles(in)
	if noSamples.Position != 0 || noSamples.Lane != "" {
		t.Errorf("player without samples should stay empty, got pos=%d lane=%q", noSamples.Position, noSamples.Lane)
	}
	if withSamples.Position != 2 {
		t.Errorf("mid pos=%d want 2", withSamples.Position)
	}
}

// ---- e2e window/dead filtering through Aggregate ----

// posEv builds an interval with parser x/y coords (chapter/128) and life state.
func posEv(t, slot int, x, y float64, life int) RawEvent {
	return RawEvent{Type: "interval", Time: t, Slot: iptr(slot), X: &x, Y: &y, LifeState: iptr(life)}
}

func TestRoleFilterWindowAndDead(t *testing.T) {
	players := []pbPlayer{
		heroPlayer(1, treant, "P0", 2),
		heroPlayer(2, rubick, "P1", 3),
	}
	evs := []RawEvent{
		epilogueEv(1, 600, 2, players),
		slotEv(0, 0),
		slotEv(1, 128),
		// pre-game (time<0) and dead (life!=0) and post-window (700) samples filtered
		posEv(-10, 0, 164.0625, 78.125, 0),
		posEv(10, 0, 164.0625, 78.125, 1),
		posEv(700, 0, 164.0625, 78.125, 0),
		// dire valid mid sample (world 16000 -> 125/128)
		posEv(20, 1, 125.0, 125.0, 0),
		statEv(1800, 1, 500, 0),
	}
	m, err := Aggregate(evs)
	if err != nil {
		t.Fatal(err)
	}
	if m.Players[0].Position != 0 || m.Players[0].Lane != "" {
		t.Errorf("radiant should have no role (all samples filtered), got pos=%d lane=%q",
			m.Players[0].Position, m.Players[0].Lane)
	}
	if m.Players[1].Position != 2 || m.Players[1].Lane != "mid" {
		t.Errorf("dire should be mid pos2, got pos=%d lane=%q", m.Players[1].Position, m.Players[1].Lane)
	}
}

// TestRoleFarmGoldPriority proves the window gold (farmGold) wins over the
// final gold (farmAll) when they disagree: slot1 is richer in the window but
// poorer overall, so with farmGold priority slot1 becomes the carry.
func TestRoleFarmGoldPriority(t *testing.T) {
	players := []pbPlayer{
		heroPlayer(1, treant, "P0", 2),
		heroPlayer(2, rubick, "P1", 2),
	}
	evs := []RawEvent{
		epilogueEv(1, 600, 2, players),
		slotEv(0, 0),
		slotEv(1, 2),
		// both safe-lane, in-window gold: slot1 (9000) beats slot0 (3000)
		goldPosEv(10, 0, 164.0625, 78.125, 3000),
		goldPosEv(10, 1, 164.0625, 78.125, 9000),
		// final gold reverses the order: slot0 (40000) over slot1 (28000)
		statEv(1800, 0, 40000, 0),
		statEv(1800, 1, 28000, 0),
	}
	m, err := Aggregate(evs)
	if err != nil {
		t.Fatal(err)
	}
	if m.Players[0].Position != 5 || m.Players[1].Position != 1 {
		t.Errorf("farmGold should win: got slot0(pos%d) slot1(pos%d), want 5,1",
			m.Players[0].Position, m.Players[1].Position)
	}
}
