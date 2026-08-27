package parser

import "strings"

var BuffCategories = map[string]string{
	// save
	"dazzle_shallow_grave":                   "save",
	"oracle_false_promise":                   "save",
	"winter_wyvern_cold_embrace":             "save",
	"omniknight_guardian_angel":              "save",
	"abaddon_borrowed_time":                  "save",
	"obsidian_destroyer_astral_imprisonment": "save",
	"pugna_decrepify":                        "save",
	"underlord_dark_rift":                    "save",
	"ringmaster_strongman_tonic":             "save",
	"tusk_snowball":                          "save",
	"item_aeon_disk":                         "save",
	"eul_cyclone":                            "save",
	"item_force_staff":                       "save",
	"item_hurricane_pike":                    "save",
	"item_pogostick_active":                  "save",
	"disperser_movespeed_buff":               "save",

	// purge
	"item_guardian_greaves":             "purge",
	"legion_commander_press_the_attack": "purge",
	"oracle_purifying_flames":           "purge",
	"omniknight_purification":           "purge",
	"item_lotus_orb_active":             "purge",

	// shield
	"abaddon_aphotic_shield":       "shield",
	"item_pavise":                  "shield",
	"lich_frost_shield":            "shield",
	"marci_bodyguard":              "shield",
	"item_pipe_of_insight":         "shield",
	"skywrath_mage_shield_barrier": "shield",

	// spell_immunity
	"omniknight_repel": "spell_immunity",

	// invisibility
	"treant_natures_guise_invis": "invisibility",
	"item_glimmer_cape_fade":     "invisibility",
	"item_shadow_amulet_fade":    "invisibility",
	"rune_invis":                 "invisibility",
	"mirana_moonlight_shadow":    "invisibility",

	// buff_stats
	"ogre_magi_bloodlust":                      "buff_stats",
	"treant_living_armor":                      "buff_stats",
	"marci_companion_run_ally_movespeed":       "buff_stats",
	"marci_bodyguarded":                        "buff_stats",
	"io_overcharge":                            "buff_stats",
	"dark_seer_surge":                          "buff_stats",
	"dark_seer_ion_shell":                      "buff_stats",
	"chen_divine_favor_armor":                  "buff_stats",
	"empower":                                  "buff_stats",
	"item_solar_crest_armor_addition":          "buff_stats",
	"item_medallion_of_courage_armor_addition": "buff_stats",
	"alchemist_berserk_potion":                 "buff_stats",

	// buff_haste
	"item_drum":        "buff_haste",
	"centaur_stampede": "buff_haste",

	// mana_restore
	"kotl_chakra_magic": "mana_restore",
}

var ItemWardCosts = map[string]int{
	"item_smoke_of_deceit": 50,
	"item_dust":            80,
}

var suffixes = []string{
	"_healing_aura", "_healing", "_aura", "_debuff",
	"_leash", "_spear", "_fade", "_buff",
	"_armor", "_active",
}

func canonicalName(inf string) string {
	for _, s := range suffixes {
		inf = strings.TrimSuffix(inf, s)
	}
	return inf
}

func classifyBreak(inflictor string) bool {
	l := strings.ToLower(inflictor)
	if l == "break" || l == "modifier_break" {
		return true
	}
	return strings.Contains(l, "silver_edge_debuff") ||
		strings.Contains(l, "vendetta_break") ||
		strings.Contains(l, "angels_demise_break") ||
		strings.Contains(l, "doom_bringer_doom_break") ||
		strings.Contains(l, "viper_strike_debuff") ||
		strings.Contains(l, "sharpshooter_debuff") ||
		strings.Contains(l, "fan_of_knives") ||
		strings.Contains(l, "berserker_troll_break")
}

func classifyFear(inflictor string) bool {
	l := strings.ToLower(inflictor)
	if strings.Contains(l, "crippling_fear") {
		return false
	}
	return strings.Contains(l, "fear") ||
		strings.Contains(l, "terrorize") ||
		strings.Contains(l, "sinister_gaze") ||
		strings.Contains(l, "savage_roar") ||
		strings.Contains(l, "dead_shot") ||
		strings.Contains(l, "tame_the_beasts") ||
		strings.Contains(l, "terror_wave") ||
		strings.Contains(l, "aether_remnant") ||
		strings.Contains(l, "will_o_wisp") ||
		strings.Contains(l, "wheel_of_wonder")
}

func classifyRoot(inflictor string) bool {
	l := strings.ToLower(inflictor)
	return strings.Contains(l, "root") ||
		strings.Contains(l, "frostbite") ||
		strings.Contains(l, "searing_chains") ||
		strings.Contains(l, "earthbind") ||
		strings.Contains(l, "ensnare") ||
		strings.Contains(l, "entangle") ||
		strings.Contains(l, "fortune") ||
		strings.Contains(l, "spinnerssnare") ||
		strings.Contains(l, "atos")
}

func classifyLeash(inflictor string) bool {
	l := strings.ToLower(inflictor)
	if strings.Contains(l, "puck_coiled") && !strings.Contains(l, "break") {
		return true
	}
	return strings.Contains(l, "pounce_leash") ||
		strings.Contains(l, "soul_chain")
}

func classifyTrap(inflictor string) bool {
	l := strings.ToLower(inflictor)
	return strings.Contains(l, "kinetic_field") ||
		strings.Contains(l, "arena_of_blood_leash")
}

func classifyTaunt(inflictor string) bool {
	l := strings.ToLower(inflictor)
	return strings.Contains(l, "berserkers_call") ||
		strings.Contains(l, "little_friends") ||
		strings.Contains(l, "winters_curse") ||
		strings.Contains(l, "legion_commander_duel") ||
		strings.Contains(l, "life_break_taunt")
}

var silenceSubstrings = []string{
	"crippling_fear",
	"static_storm",
	"ancient_seal",
	"doom_bringer_doom",
	"smoke_screen",
	"geomagnetic_grip_debuff",
	"bloodthorn",
	"orchid",
	"ink_creature_debuff",
	"flux_alone",
	"inner_fire_disarm",
	"silencer_last_word",
}

func classifySilence(inflictor string) bool {
	l := strings.ToLower(inflictor)
	if strings.Contains(l, "oppresive") {
		return false
	}
	if strings.HasSuffix(l, "_knockback") || strings.Contains(l, "silencer_last_word_disarm") {
		return false
	}
	if strings.Contains(strings.ReplaceAll(l, "_silencer", ""), "_silence") {
		return true
	}
	for _, s := range silenceSubstrings {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

var InstantHealItems = map[string]bool{
	"item_holy_locket":     true,
	"item_famango":         true,
	"item_great_famango":   true,
	"item_greater_famango": true,
}

func classifyHeal(inflictor string) bool {
	l := strings.ToLower(inflictor)
	return l == "modifier_item_urn_heal" ||
		l == "modifier_item_spirit_vessel_heal" ||
		l == "modifier_item_essence_distiller_heal" ||
		l == "modifier_bottle_regeneration" ||
		l == "modifier_item_polliwog_charm_buff"
}

func classifyDisarm(inflictor string) bool {
	l := strings.ToLower(inflictor)
	if strings.Contains(l, "lucky_shot") ||
		strings.Contains(l, "last_word") ||
		strings.Contains(l, "inner_fire") ||
		strings.Contains(l, "scatterblast") {
		return false
	}
	return strings.Contains(l, "disarm") ||
		strings.Contains(l, "heavens_halberd") ||
		strings.Contains(l, "concussive_grenade") ||
		strings.Contains(l, "fates_edict")
}
