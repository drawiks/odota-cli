package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type playerInfo struct {
	Slot    int
	SteamID int64
	Hero    string
	HeroID  int
	Name    string
	Team    string
}

func Aggregate(events []RawEvent) (*Match, error) {
	m := &Match{}

	var players []playerInfo
	slotToHero := map[int]string{}
	lastInterval := map[int]RawEvent{}
	lastIntervalTime := 0

	slotTeam := map[int]string{}
	heroTeam := map[string]string{}

	healingByAttacker := map[string]int{}
	heroDamageByAttacker := map[string]int{}
	damageTakenByTarget := map[string]int{}
	towerDamageByAttacker := map[string]int{}

	type stunAgg struct {
		Total float64
	}
	stunSources := map[string]stunAgg{}

	type buffAgg struct {
		Value float64
	}
	buffSources := map[string]buffAgg{}

	type activeBuff struct {
		Inflictor string
		Category  string
		AddTime   int
	}
	activeBuffs := map[string][]activeBuff{}

	type activeFear struct {
		Inflictor string
		AddTime   int
	}
	activeFears := map[string][]activeFear{}
	fearSources := map[string]float64{}

	type activeRoot struct {
		Inflictor string
		AddTime   int
	}
	activeRoots := map[string][]activeRoot{}
	rootSources := map[string]float64{}

	type activeLeash struct {
		Inflictor string
		AddTime   int
	}
	activeLeashes := map[string][]activeLeash{}
	leashSources := map[string]float64{}

	type activeTrap struct {
		Inflictor string
		AddTime   int
	}
	activeTraps := map[string][]activeTrap{}
	trapSources := map[string]float64{}

	type activeTaunt struct {
		Inflictor string
		AddTime   int
	}
	activeTaunts := map[string][]activeTaunt{}
	tauntSources := map[string]float64{}

	type activeSilence struct {
		Inflictor string
		AddTime   int
	}
	activeSilences := map[string][]activeSilence{}
	silenceSources := map[string]float64{}

	type activeBreak struct {
		Inflictor string
		AddTime   int
	}
	activeBreaks := map[string][]activeBreak{}
	breakSources := map[string]float64{}

	type activeDisarm struct {
		Inflictor string
		AddTime   int
	}
	activeDisarms := map[string][]activeDisarm{}
	disarmSources := map[string]float64{}

	type activeHeal struct {
		Inflictor string
		AddTime   int
	}
	activeHeals := map[string][]activeHeal{}

	type healAgg struct {
		Duration float64
		Value    float64
	}
	healSources := map[string]healAgg{}

	goldSpentSmoke := map[string]int{}
	goldSpentDust := map[string]int{}

	firstBloodSlot := -1

	heroIsAlly := func(a, b string) bool {
		return isAllyByTeam(heroTeam, a, b)
	}

	for _, e := range events {
		switch e.Type {

		case "epilogue":
			var keyStr string
			if err := json.Unmarshal(e.Key, &keyStr); err != nil {
				return nil, fmt.Errorf("parse epilogue key: %w", err)
			}
			parsed, err := parseEpilogue(keyStr)
			if err != nil {
				return nil, fmt.Errorf("parse epilogue: %w", err)
			}
			m.MatchID = parsed.MatchID
			m.DurationSec = parsed.DurationSec
			m.RadiantWin = parsed.RadiantWin
			players = parsed.Players

		case "interval":
			if e.Slot != nil {
				lastInterval[*e.Slot] = e
				if e.Time > lastIntervalTime {
					lastIntervalTime = e.Time
				}
				if e.Unit != "" {
					hero := unitToHeroName(e.Unit)
					slotToHero[*e.Slot] = hero
					if team, ok := slotTeam[*e.Slot]; ok {
						heroTeam[heroKey(hero)] = team
					}
				}
			}

		case "player_slot":
			if len(e.Key) > 0 && len(e.Value) > 0 {
				var slotStr string
				if err := json.Unmarshal(e.Key, &slotStr); err != nil {
					continue
				}
				slot := 0
				if _, err := fmt.Sscanf(slotStr, "%d", &slot); err != nil {
					continue
				}
				val := parseRawFloat(e.Value)
				isDire := int(val)&128 != 0
				team := "radiant"
				if isDire {
					team = "dire"
				}
				slotTeam[slot] = team
				if hero, ok := slotToHero[slot]; ok {
					heroTeam[heroKey(hero)] = team
				}
				if slot < len(players) {
					players[slot].Team = team
				}
			}

		case "DOTA_COMBATLOG_HEAL":
			if e.Attackerhero == nil || !*e.Attackerhero {
				continue
			}
			if e.Targethero == nil || !*e.Targethero {
				continue
			}
			attacker := e.AttackerName
			target := e.TargetName
			if attacker == "" || target == "" || attacker == target {
				continue
			}
			if !heroIsAlly(attacker, target) {
				continue
			}
			healingByAttacker[heroKey(attacker)] += int(parseRawFloat(e.Value))

			if InstantHealItems[e.Inflictor] {
				skey := heroKey(attacker) + ":" + e.Inflictor
				agg := healSources[skey]
				agg.Value += parseRawFloat(e.Value)
				healSources[skey] = agg
			}

		case "DOTA_COMBATLOG_DAMAGE":
			key := e.AttackerName
			if strings.HasPrefix(e.Sourcename, "npc_dota_hero_") {
				key = e.Sourcename
			}
			val := int(parseRawFloat(e.Value))
			if e.Targethero != nil && *e.Targethero && !isIllusion(e.Targetillusion) &&
				key != e.TargetName {
				heroDamageByAttacker[heroKey(key)] += val
			}
			if e.Attackerhero != nil && *e.Attackerhero && !isIllusion(e.Attackerillusion) &&
				isBuilding(e.TargetName) {
				towerDamageByAttacker[heroKey(key)] += val
			}
			if e.Targethero != nil && *e.Targethero && !isIllusion(e.Targetillusion) &&
				e.Attackerhero != nil && *e.Attackerhero && key != e.TargetName {
				damageTakenByTarget[heroKey(e.TargetName)] += int(parseRawFloat(e.Value))
			}

		case "DOTA_COMBATLOG_MODIFIER_ADD":
			if e.Targethero == nil || !*e.Targethero {
				continue
			}
			if e.Targetillusion != nil && *e.Targetillusion {
				continue
			}
			inf := e.Inflictor
			if inf == "" {
				continue
			}
			attacker := e.AttackerName
			if attacker == "" {
				attacker = e.TargetName
			}
			target := e.TargetName
			if attacker == target {
				continue
			}
			clean := canonicalName(cleanInflictor(inf))
			if classifyBreak(inf) {
				if !heroIsAlly(attacker, target) {
					bKey := attacker + ":" + target + ":" + clean
					activeBreaks[bKey] = append(activeBreaks[bKey], activeBreak{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyFear(inf) {
				if !heroIsAlly(attacker, target) {
					fKey := attacker + ":" + target + ":" + clean
					activeFears[fKey] = append(activeFears[fKey], activeFear{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyRoot(inf) {
				if !heroIsAlly(attacker, target) {
					rKey := attacker + ":" + target + ":" + clean
					activeRoots[rKey] = append(activeRoots[rKey], activeRoot{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyLeash(inf) {
				if !heroIsAlly(attacker, target) {
					lKey := attacker + ":" + target + ":" + clean
					activeLeashes[lKey] = append(activeLeashes[lKey], activeLeash{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyTrap(inf) {
				if !heroIsAlly(attacker, target) {
					tKey := attacker + ":" + target + ":" + clean
					activeTraps[tKey] = append(activeTraps[tKey], activeTrap{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyTaunt(inf) {
				if !heroIsAlly(attacker, target) {
					tKey := attacker + ":" + target + ":" + clean
					activeTaunts[tKey] = append(activeTaunts[tKey], activeTaunt{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifySilence(inf) {
				if !heroIsAlly(attacker, target) {
					silKey := attacker + ":" + target + ":" + clean
					activeSilences[silKey] = append(activeSilences[silKey], activeSilence{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyDisarm(inf) {
				if !heroIsAlly(attacker, target) {
					dKey := attacker + ":" + target + ":" + clean
					activeDisarms[dKey] = append(activeDisarms[dKey], activeDisarm{
						Inflictor: clean,
						AddTime:   e.Time,
					})
				}
				continue
			}
			if classifyHeal(inf) {
				hKey := attacker + ":" + target + ":" + clean
				activeHeals[hKey] = append(activeHeals[hKey], activeHeal{
					Inflictor: clean,
					AddTime:   e.Time,
				})
				continue
			}
			cat := lookupCategory(inf)
			if cat == "" {
				continue
			}
			if cat != "save" && cat != "purge" && cat != "shield" && cat != "spell_immunity" &&
				cat != "invisibility" && cat != "buff_stats" && cat != "buff_haste" && cat != "mana_restore" {
				continue
			}
			if !heroIsAlly(attacker, target) {
				continue
			}
			abKey := attacker + ":" + target + ":" + clean
			activeBuffs[abKey] = append(activeBuffs[abKey], activeBuff{
				Inflictor: clean,
				Category:  cat,
				AddTime:   e.Time,
			})
			if e.StunDuration != nil && *e.StunDuration > 0 {
				skey := fmt.Sprintf("%s:%s", attacker, clean)
				agg := stunSources[skey]
				agg.Total += *e.StunDuration
				stunSources[skey] = agg
			}

		case "DOTA_COMBATLOG_MODIFIER_REMOVE":
			if e.Targethero == nil || !*e.Targethero {
				continue
			}
			if e.Targetillusion != nil && *e.Targetillusion {
				continue
			}
			inf := e.Inflictor
			if inf == "" {
				continue
			}
			clean := canonicalName(cleanInflictor(inf))
			attacker := e.AttackerName
			if attacker == "" {
				attacker = e.TargetName
			}
			target := e.TargetName
			if attacker == target {
				continue
			}
			if classifyBreak(inf) {
				if !heroIsAlly(attacker, target) {
					bKey := attacker + ":" + target + ":" + clean
					breaks := activeBreaks[bKey]
					for i := len(breaks) - 1; i >= 0; i-- {
						if breaks[i].Inflictor == clean {
							dur := float64(e.Time - breaks[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								breakSources[skey] += dur
							}
							activeBreaks[bKey] = append(breaks[:i], breaks[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyFear(inf) {
				if !heroIsAlly(attacker, target) {
					fKey := attacker + ":" + target + ":" + clean
					fears := activeFears[fKey]
					if len(fears) > 0 {
						for i := len(fears) - 1; i >= 0; i-- {
							if fears[i].Inflictor == clean {
								dur := float64(e.Time - fears[i].AddTime)
								if dur > 0 {
									skey := attacker + ":" + clean
									fearSources[skey] += dur
								}
								activeFears[fKey] = append(fears[:i], fears[i+1:]...)
								break
							}
						}
					}
				}
				continue
			}
			if classifyRoot(inf) {
				if !heroIsAlly(attacker, target) {
					rKey := attacker + ":" + target + ":" + clean
					roots := activeRoots[rKey]
					for i := len(roots) - 1; i >= 0; i-- {
						if roots[i].Inflictor == clean {
							dur := float64(e.Time - roots[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								rootSources[skey] += dur
							}
							activeRoots[rKey] = append(roots[:i], roots[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyLeash(inf) {
				if !heroIsAlly(attacker, target) {
					lKey := attacker + ":" + target + ":" + clean
					leashes := activeLeashes[lKey]
					for i := len(leashes) - 1; i >= 0; i-- {
						if leashes[i].Inflictor == clean {
							dur := float64(e.Time - leashes[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								leashSources[skey] += dur
							}
							activeLeashes[lKey] = append(leashes[:i], leashes[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyTrap(inf) {
				if !heroIsAlly(attacker, target) {
					tKey := attacker + ":" + target + ":" + clean
					traps := activeTraps[tKey]
					for i := len(traps) - 1; i >= 0; i-- {
						if traps[i].Inflictor == clean {
							dur := float64(e.Time - traps[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								trapSources[skey] += dur
							}
							activeTraps[tKey] = append(traps[:i], traps[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyTaunt(inf) {
				if !heroIsAlly(attacker, target) {
					tKey := attacker + ":" + target + ":" + clean
					taunts := activeTaunts[tKey]
					for i := len(taunts) - 1; i >= 0; i-- {
						if taunts[i].Inflictor == clean {
							dur := float64(e.Time - taunts[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								tauntSources[skey] += dur
							}
							activeTaunts[tKey] = append(taunts[:i], taunts[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifySilence(inf) {
				if !heroIsAlly(attacker, target) {
					silKey := attacker + ":" + target + ":" + clean
					silences := activeSilences[silKey]
					for i := len(silences) - 1; i >= 0; i-- {
						if silences[i].Inflictor == clean {
							dur := float64(e.Time - silences[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								silenceSources[skey] += dur
							}
							activeSilences[silKey] = append(silences[:i], silences[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyDisarm(inf) {
				if !heroIsAlly(attacker, target) {
					dKey := attacker + ":" + target + ":" + clean
					disarms := activeDisarms[dKey]
					for i := len(disarms) - 1; i >= 0; i-- {
						if disarms[i].Inflictor == clean {
							dur := float64(e.Time - disarms[i].AddTime)
							if dur > 0 {
								skey := attacker + ":" + clean
								disarmSources[skey] += dur
							}
							activeDisarms[dKey] = append(disarms[:i], disarms[i+1:]...)
							break
						}
					}
				}
				continue
			}
			if classifyHeal(inf) {
				hKey := attacker + ":" + target + ":" + clean
				heals := activeHeals[hKey]
				for i := len(heals) - 1; i >= 0; i-- {
					if heals[i].Inflictor == clean {
						dur := float64(e.Time - heals[i].AddTime)
						if dur > 0 {
							skey := attacker + ":" + clean
							agg := healSources[skey]
							agg.Duration += dur
							healSources[skey] = agg
						}
						activeHeals[hKey] = append(heals[:i], heals[i+1:]...)
						break
					}
				}
				continue
			}
			if !heroIsAlly(attacker, target) {
				continue
			}
			abKey := attacker + ":" + target + ":" + clean
			buffs := activeBuffs[abKey]
			if len(buffs) == 0 {
				continue
			}
			for i := len(buffs) - 1; i >= 0; i-- {
				if buffs[i].Inflictor == clean {
					duration := float64(e.Time - buffs[i].AddTime)
					if duration > 0 {
						cat := buffs[i].Category
						bkey := attacker + ":" + cat + ":" + clean
						agg := buffSources[bkey]
						agg.Value += duration
						buffSources[bkey] = agg
					}
					activeBuffs[abKey] = append(buffs[:i], buffs[i+1:]...)
					break
				}
			}

		case "DOTA_COMBATLOG_PURCHASE":
			vname := e.Valuename
			if vname == "" {
				continue
			}
			hero := e.TargetName
			cost, ok := ItemWardCosts[vname]
			if !ok {
				continue
			}
			switch {
			case strings.Contains(vname, "smoke"):
				goldSpentSmoke[heroKey(hero)] += cost
			case strings.Contains(vname, "dust"):
				goldSpentDust[heroKey(hero)] += cost
			}

		case "CHAT_MESSAGE_FIRSTBLOOD":
			if len(e.Player1) > 0 {
				var p1 float64
				if err := json.Unmarshal(e.Player1, &p1); err == nil {
					firstBloodSlot = int(p1)
				}
			}
		}
	}

	for i := range players {
		p := &players[i]
		if hero, ok := slotToHero[p.Slot]; ok {
			p.Hero = hero
			p.HeroID = HeroMap[hero]
		}
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].Slot < players[j].Slot
	})

	m.Players = make([]Player, len(players))
	for i, p := range players {
		mp := &m.Players[i]
		mp.SteamID = p.SteamID
		mp.PlayerID = p.Slot
		mp.HeroID = p.HeroID
		mp.Hero = strings.TrimPrefix(p.Hero, "npc_dota_hero_")
		mp.Team = p.Team
		mp.Name = p.Name

		if iv, ok := lastInterval[p.Slot]; ok {
			mp.Kills = iv.Kills
			mp.Deaths = iv.Deaths
			mp.Assists = iv.Assists
			mp.Level = iv.Level
			mp.LastHits = iv.Lh
			mp.Networth = iv.Networth
			mp.StunDuration = iv.Stuns
			mp.CampsStacked = iv.CampsStacked
			mp.CreepsStacked = iv.CreepsStacked
			mp.RunePickups = iv.RunePickups
			mp.FirstBlood = firstBloodSlot == p.Slot

			if lastIntervalTime > 0 {
				gameMinutes := float64(lastIntervalTime) / 60.0
				mp.GPM = int(math.Round(float64(iv.Gold) / gameMinutes))
				mp.XPM = int(math.Round(float64(iv.Xp) / gameMinutes))
			}
			mp.GoldSpentWards = iv.SenPlaced * 50
		}

		pHeroKey := heroKey(p.Hero)
		mp.Healing = healingByAttacker[pHeroKey]
		mp.HeroDamage = heroDamageByAttacker[pHeroKey]
		mp.DamageTaken = damageTakenByTarget[pHeroKey]
		mp.TowerDamage = towerDamageByAttacker[pHeroKey]

		mp.GoldSpentSmoke = goldSpentSmoke[pHeroKey]
		mp.GoldSpentDust = goldSpentDust[pHeroKey]

		for key, agg := range stunSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.StunSources = append(mp.StunSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "stun",
					Duration:  agg.Total,
				})
			}
		}
		sort.Slice(mp.StunSources, func(i, j int) bool {
			if mp.StunSources[i].Duration != mp.StunSources[j].Duration {
				return mp.StunSources[i].Duration > mp.StunSources[j].Duration
			}
			return mp.StunSources[i].Inflictor < mp.StunSources[j].Inflictor
		})

		var totalBuffDuration float64
		for key, agg := range buffSources {
			parts := strings.SplitN(key, ":", 3)
			if len(parts) == 3 && heroKey(parts[0]) == pHeroKey {
				cat := parts[1]
				inf := parts[2]
				entry := SourceEntry{
					Inflictor: inf,
					Category:  cat,
					Duration:  math.Round(agg.Value),
				}
				totalBuffDuration += agg.Value
				mp.BuffSources = append(mp.BuffSources, entry)
			}
		}
		sort.Slice(mp.BuffSources, func(i, j int) bool {
			if mp.BuffSources[i].Duration != mp.BuffSources[j].Duration {
				return mp.BuffSources[i].Duration > mp.BuffSources[j].Duration
			}
			return mp.BuffSources[i].Inflictor < mp.BuffSources[j].Inflictor
		})

		mp.BuffDuration = math.Round(totalBuffDuration)

		var totalFearDuration float64
		for key, dur := range fearSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.FearSources = append(mp.FearSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "fear",
					Duration:  math.Round(dur),
				})
				totalFearDuration += dur
			}
		}
		sort.Slice(mp.FearSources, func(i, j int) bool {
			if mp.FearSources[i].Duration != mp.FearSources[j].Duration {
				return mp.FearSources[i].Duration > mp.FearSources[j].Duration
			}
			return mp.FearSources[i].Inflictor < mp.FearSources[j].Inflictor
		})
		mp.FearDuration = math.Round(totalFearDuration)

		var totalRootDuration float64
		for key, dur := range rootSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.RootSources = append(mp.RootSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "root",
					Duration:  math.Round(dur),
				})
				totalRootDuration += dur
			}
		}
		sort.Slice(mp.RootSources, func(i, j int) bool {
			if mp.RootSources[i].Duration != mp.RootSources[j].Duration {
				return mp.RootSources[i].Duration > mp.RootSources[j].Duration
			}
			return mp.RootSources[i].Inflictor < mp.RootSources[j].Inflictor
		})
		mp.RootsDuration = math.Round(totalRootDuration)

		var totalLeashDuration float64
		for key, dur := range leashSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.LeashSources = append(mp.LeashSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "leash",
					Duration:  math.Round(dur),
				})
				totalLeashDuration += dur
			}
		}
		sort.Slice(mp.LeashSources, func(i, j int) bool {
			if mp.LeashSources[i].Duration != mp.LeashSources[j].Duration {
				return mp.LeashSources[i].Duration > mp.LeashSources[j].Duration
			}
			return mp.LeashSources[i].Inflictor < mp.LeashSources[j].Inflictor
		})
		mp.LeashDuration = math.Round(totalLeashDuration)

		var totalTrapDuration float64
		for key, dur := range trapSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.TrapSources = append(mp.TrapSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "trap",
					Duration:  math.Round(dur),
				})
				totalTrapDuration += dur
			}
		}
		sort.Slice(mp.TrapSources, func(i, j int) bool {
			if mp.TrapSources[i].Duration != mp.TrapSources[j].Duration {
				return mp.TrapSources[i].Duration > mp.TrapSources[j].Duration
			}
			return mp.TrapSources[i].Inflictor < mp.TrapSources[j].Inflictor
		})
		mp.TrapDuration = math.Round(totalTrapDuration)

		var totalTauntDuration float64
		for key, dur := range tauntSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.TauntSources = append(mp.TauntSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "taunt",
					Duration:  math.Round(dur),
				})
				totalTauntDuration += dur
			}
		}
		sort.Slice(mp.TauntSources, func(i, j int) bool {
			if mp.TauntSources[i].Duration != mp.TauntSources[j].Duration {
				return mp.TauntSources[i].Duration > mp.TauntSources[j].Duration
			}
			return mp.TauntSources[i].Inflictor < mp.TauntSources[j].Inflictor
		})
		mp.TauntDuration = math.Round(totalTauntDuration)

		var totalSilenceDuration float64
		for key, dur := range silenceSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.SilenceSources = append(mp.SilenceSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "silence",
					Duration:  math.Round(dur),
				})
				totalSilenceDuration += dur
			}
		}
		sort.Slice(mp.SilenceSources, func(i, j int) bool {
			if mp.SilenceSources[i].Duration != mp.SilenceSources[j].Duration {
				return mp.SilenceSources[i].Duration > mp.SilenceSources[j].Duration
			}
			return mp.SilenceSources[i].Inflictor < mp.SilenceSources[j].Inflictor
		})
		mp.SilenceDuration = math.Round(totalSilenceDuration)

		var totalBreakDuration float64
		for key, dur := range breakSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.BreakSources = append(mp.BreakSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "break",
					Duration:  math.Round(dur),
				})
				totalBreakDuration += dur
			}
		}
		sort.Slice(mp.BreakSources, func(i, j int) bool {
			if mp.BreakSources[i].Duration != mp.BreakSources[j].Duration {
				return mp.BreakSources[i].Duration > mp.BreakSources[j].Duration
			}
			return mp.BreakSources[i].Inflictor < mp.BreakSources[j].Inflictor
		})
		mp.BreakDuration = math.Round(totalBreakDuration)

		var totalDisarmDuration float64
		for key, dur := range disarmSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				mp.DisarmSources = append(mp.DisarmSources, SourceEntry{
					Inflictor: parts[1],
					Category:  "disarm",
					Duration:  math.Round(dur),
				})
				totalDisarmDuration += dur
			}
		}
		sort.Slice(mp.DisarmSources, func(i, j int) bool {
			if mp.DisarmSources[i].Duration != mp.DisarmSources[j].Duration {
				return mp.DisarmSources[i].Duration > mp.DisarmSources[j].Duration
			}
			return mp.DisarmSources[i].Inflictor < mp.DisarmSources[j].Inflictor
		})
		mp.DisarmDuration = math.Round(totalDisarmDuration)

		var totalHealDuration float64
		var totalHealValue float64
		for key, agg := range healSources {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 && heroKey(parts[0]) == pHeroKey {
				entry := SourceEntry{
					Inflictor: parts[1],
					Category:  "heal",
				}
				if agg.Duration > 0 {
					entry.Duration = math.Round(agg.Duration)
					totalHealDuration += agg.Duration
				}
				if agg.Value > 0 {
					entry.Value = math.Round(agg.Value)
					totalHealValue += agg.Value
				}
				mp.HealSources = append(mp.HealSources, entry)
			}
		}
		sort.Slice(mp.HealSources, func(i, j int) bool {
			if mp.HealSources[i].Duration != mp.HealSources[j].Duration {
				return mp.HealSources[i].Duration > mp.HealSources[j].Duration
			}
			if mp.HealSources[i].Value != mp.HealSources[j].Value {
				return mp.HealSources[i].Value > mp.HealSources[j].Value
			}
			return mp.HealSources[i].Inflictor < mp.HealSources[j].Inflictor
		})

		mp.HealDuration = math.Round(totalHealDuration)
		mp.HealValue = math.Round(totalHealValue)
	}

	return m, nil
}

type epilogueResult struct {
	MatchID     int64
	DurationSec float64
	RadiantWin  bool
	Players     []playerInfo
}

func parseEpilogue(keyJSON string) (*epilogueResult, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(keyJSON), &raw); err != nil {
		return nil, err
	}

	duration := parseRawFloat(raw["playbackTime_"])

	giRaw, ok := raw["gameInfo_"]
	if !ok {
		return nil, fmt.Errorf("gameInfo_ not found")
	}

	var gi struct {
		Dota_ struct {
			MatchId_    float64 `json:"matchId_"`
			GameWinner_ int     `json:"gameWinner_"`
			PlayerInfo_ []struct {
				Steamid_  json.RawMessage `json:"steamid_"`
				HeroName_ struct {
					Bytes []int `json:"bytes"`
				} `json:"heroName_"`
				PlayerName_ struct {
					Bytes []int `json:"bytes"`
				} `json:"playerName_"`
				GameTeam_ int `json:"gameTeam_"`
			} `json:"playerInfo_"`
		} `json:"dota_"`
	}
	if err := json.Unmarshal(giRaw, &gi); err != nil {
		return nil, fmt.Errorf("parse gameInfo_: %w", err)
	}

	r := &epilogueResult{
		MatchID:     int64(gi.Dota_.MatchId_),
		DurationSec: duration,
		RadiantWin:  gi.Dota_.GameWinner_ == 2,
	}

	for i, pi := range gi.Dota_.PlayerInfo_ {
		hero := intsToTrimmedString(pi.HeroName_.Bytes)
		name := intsToTrimmedString(pi.PlayerName_.Bytes)
		steamID := parseInt64(pi.Steamid_)
		team := "radiant"
		if pi.GameTeam_ == 3 {
			team = "dire"
		}
		r.Players = append(r.Players, playerInfo{
			Slot:    i,
			SteamID: steamID,
			Hero:    hero,
			HeroID:  HeroMap[hero],
			Name:    name,
			Team:    team,
		})
	}

	return r, nil
}

func heroKey(s string) string {
	return strings.ReplaceAll(s, "_", "")
}

func isAllyByTeam(heroTeam map[string]string, a, b string) bool {
	ta, okA := heroTeam[heroKey(a)]
	tb, okB := heroTeam[heroKey(b)]
	if !okA || !okB {
		return false
	}
	return ta == tb
}

func cleanInflictor(inf string) string {
	return strings.TrimPrefix(inf, "modifier_")
}

func lookupCategory(inf string) string {
	if cat, ok := BuffCategories[cleanInflictor(inf)]; ok {
		return cat
	}
	return ""
}

func intsToTrimmedString(ints []int) string {
	b := make([]byte, len(ints))
	for i, v := range ints {
		b[i] = byte(v)
	}
	return strings.TrimRight(string(b), "\x00")
}

var camelCaseRe = regexp.MustCompile(`([a-z])([A-Z])`)

func unitToHeroName(unit string) string {
	const prefix = "CDOTA_Unit_Hero_"
	if !strings.HasPrefix(unit, prefix) {
		return unit
	}
	name := unit[len(prefix):]
	name = camelCaseRe.ReplaceAllString(name, "${1}_${2}")
	return "npc_dota_hero_" + strings.ToLower(name)
}

func parseInt64(raw json.RawMessage) int64 {
	s := strings.TrimSpace(string(raw))
	if len(s) == 0 {
		return 0
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return int64(n)
	}
	var st string
	if err := json.Unmarshal(raw, &st); err == nil {
		if n, err := strconv.ParseInt(strings.TrimSpace(st), 10, 64); err == nil {
			return n
		}
		var nn float64
		if _, err := fmt.Sscanf(st, "%f", &nn); err == nil {
			return int64(nn)
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		var high, low uint64
		if h, ok := obj["high"]; ok {
			json.Unmarshal(h, &high)
		}
		if l, ok := obj["low"]; ok {
			json.Unmarshal(l, &low)
		}
		if high > 0 || low > 0 {
			return int64(high<<32 | low)
		}
	}
	return 0
}

func parseRawFloat(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	return 0
}

func isIllusion(b *bool) bool {
	return b != nil && *b
}

func isBuilding(name string) bool {
	return strings.Contains(name, "tower") ||
		strings.Contains(name, "rax") ||
		strings.Contains(name, "fortress") ||
		strings.Contains(name, "building") ||
		strings.Contains(name, "shrine") ||
		strings.Contains(name, "effigy")
}
