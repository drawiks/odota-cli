package parser

import "sort"

// Lane/position detection assigns each player a position (1-5) and lane label
// from where their hero stood during the first roleWindowSec seconds of game
// time. The parser's m_iSuggestedLanes is not used: it is the pre-game
// expectation, which frequently differs from the lanes heroes actually play.
//
// Geometry: gem-dota world-coordinate lane zones, no mirroring (the map runs
// radiant base SW -> dire base NE, the diagonal is mid). A zone's meaning
// depends on the team, so the zone is converted to a lane per team at
// assignment time. The parser's interval x/y are already (cell*128+vec)/128,
// so world units are x*128; dead heroes (life_state != 0) are skipped so the
// lane histogram is not skewed.

const (
	roleWindowSec = 600.0

	// midBand is the half-width of the mid diagonal band.
	midBand = 2000.0
	// midXMin/midXMax bound the diagonal box.
	midXMin = 10500.0
	midXMax = 22000.0
	// safeRY/safeRX bound the bottom-right corner: radiant safe, dire off.
	safeRY = 12500.0
	safeRX = 20000.0
	// offRX/offRY bound the top-left corner: radiant off, dire safe.
	offRX = 12500.0
	offRY = 19000.0

	// zoneDominanceFrac is the minimum share of a hero's samples in a single
	// zone for that zone to be considered the hero's lane; below it the hero
	// is treated as roaming (pos 4).
	zoneDominanceFrac = 0.45
)

// roleZone is a world-coordinate lane zone.
type roleZone int

const (
	zoneOther roleZone = iota
	zoneMid
	zoneSafe
	zoneOff
)

// classifyZone maps a world position to a lane zone.
func classifyZone(x, y float64) roleZone {
	if x > midXMin-midBand && x < midXMax+midBand && y > midXMin-midBand && y < midXMax+midBand {
		if dx := x - y; dx > -midBand && dx < midBand {
			return zoneMid
		}
	}
	if y < safeRY && x > safeRX {
		return zoneSafe
	}
	if x < offRX && y > offRY {
		return zoneOff
	}
	return zoneOther
}

// laneKind is a team-relative lane.
type laneKind int

const (
	laneUnknown laneKind = iota
	laneMid
	laneSafe
	laneOff
)

// roleStats accumulates a hero's zone samples.
type roleStats struct {
	zones [4]int
	total int
}

func (r *roleStats) sample(x, y float64) {
	r.zones[classifyZone(x, y)]++
	r.total++
}

// dominant returns the dominant zone and whether it clears the threshold.
func (r *roleStats) dominant() (roleZone, bool) {
	if r.total == 0 {
		return zoneOther, false
	}
	best, bestN := zoneOther, 0
	for z := zoneMid; z <= zoneOff; z++ {
		if r.zones[z] > bestN {
			best, bestN = z, r.zones[z]
		}
	}
	return best, float64(bestN) >= zoneDominanceFrac*float64(r.total)
}

// laneOf converts the dominant zone to the team-relative lane.
func laneOf(z roleZone, team string) laneKind {
	switch z {
	case zoneMid:
		return laneMid
	case zoneSafe:
		if team == "dire" {
			return laneOff
		}
		return laneSafe
	case zoneOff:
		if team == "dire" {
			return laneSafe
		}
		return laneOff
	}
	return laneUnknown
}

// laneName maps a position to the output lane label.
func laneName(pos int) string {
	switch pos {
	case 1:
		return "safe"
	case 2:
		return "mid"
	case 3:
		return "off"
	case 4:
		return "supp"
	case 5:
		return "hard_supp"
	}
	return ""
}

// roleInput ties one built Player to its role samples and farm signal.
type roleInput struct {
	p        *Player
	stats    *roleStats
	farmGold int // gold earned within the role window
	farmAll  int // total gold earned (fallback when the window is empty)
}

func (r roleInput) farm() int {
	if r.farmGold != 0 {
		return r.farmGold
	}
	return r.farmAll
}

// assignRoles computes per-team positions from role samples and writes them
// back into the players. Players without samples (no position data) keep an
// empty position/lane. The 2-1-2 heuristic: mid -> 2; the safe pair by farm
// -> 1 (carry) / 5 (hard support); the off pair by farm -> 3 (off core) /
// 4 (soft support); roamers and overfull lanes fill the free positions by
// farm, preferring 4 (supp).
func assignRoles(in []roleInput) {
	radiant := make([]roleInput, 0, 5)
	dire := make([]roleInput, 0, 5)
	for _, r := range in {
		if r.stats == nil || r.stats.total == 0 {
			continue
		}
		switch r.p.Team {
		case "radiant":
			radiant = append(radiant, r)
		case "dire":
			dire = append(dire, r)
		}
	}
	assignTeam(radiant)
	assignTeam(dire)
}

func assignTeam(in []roleInput) {
	var mids, safes, offs, unk []roleInput
	for _, r := range in {
		if z, ok := r.stats.dominant(); ok {
			switch laneOf(z, r.p.Team) {
			case laneMid:
				mids = append(mids, r)
			case laneSafe:
				safes = append(safes, r)
			case laneOff:
				offs = append(offs, r)
			default:
				unk = append(unk, r)
			}
		} else {
			unk = append(unk, r)
		}
	}
	byFarm := func(s []roleInput) {
		sort.SliceStable(s, func(i, j int) bool { return s[i].farm() > s[j].farm() })
	}
	byFarm(mids)
	byFarm(safes)
	byFarm(offs)
	byFarm(unk)

	used := map[int]bool{}
	set := func(r roleInput, pos int) {
		r.p.Position = pos
		r.p.Lane = laneName(pos)
		used[pos] = true
	}
	for _, r := range mids {
		if !used[laneMidPos] {
			set(r, laneMidPos)
		} else {
			unk = append(unk, r)
		}
	}
	for _, r := range safes {
		if !used[safeCarryPos] {
			set(r, safeCarryPos)
		} else if !used[safeSuppPos] {
			set(r, safeSuppPos)
		} else {
			unk = append(unk, r)
		}
	}
	for _, r := range offs {
		if !used[offCorePos] {
			set(r, offCorePos)
		} else if !used[offSuppPos] {
			set(r, offSuppPos)
		} else {
			unk = append(unk, r)
		}
	}
	// Roamers, junglers and overfull lanes fill the free positions in a
	// support-first order, so the richest leftover is not a pos-5.
	free := make([]int, 0, 5)
	for _, p := range []int{offSuppPos, safeSuppPos, safeCarryPos, offCorePos, laneMidPos} {
		if !used[p] {
			free = append(free, p)
		}
	}
	for i, r := range unk {
		if i < len(free) {
			set(r, free[i])
		}
	}
}

// Position constants (1-5).
const (
	safeCarryPos = 1
	laneMidPos   = 2
	offCorePos   = 3
	offSuppPos   = 4
	safeSuppPos  = 5
)
