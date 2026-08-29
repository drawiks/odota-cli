package parser

import "testing"

func TestCanonicalName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"healing_aura", "x_healing_aura", "x"},
		{"healing", "x_healing", "x"},
		{"aura", "x_aura", "x"},
		{"debuff", "x_debuff", "x"},
		{"leash", "x_leash", "x"},
		{"spear", "x_spear", "x"},
		{"fade", "x_fade", "x"},
		{"buff", "x_buff", "x"},
		{"armor", "x_armor", "x"},
		{"active", "x_active", "x"},
		{"all that match in list order stripped", "item_glimmer_cape_fade_healing_aura", "item_glimmer_cape"},
		{"no suffix unchanged", "dazzle_shallow_grave", "dazzle_shallow_grave"},
		{"empty", "", ""},
		{"one pass only: trailing suffix stripped first", "x_aura_buff", "x_aura"},
		{"leash stripped for trap", "modifier_mars_arena_of_blood_leash", "modifier_mars_arena_of_blood"},
		{"debuff stripped for halberd", "modifier_heavens_halberd_debuff", "modifier_heavens_halberd"},
		{"aura stripped for little_friends", "modifier_enchantress_little_friends_aura", "modifier_enchantress_little_friends"},
		{"substring suffix not stripped mid-word", "x_ausocial", "x_ausocial"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canonicalName(tt.in); got != tt.want {
				t.Errorf("canonicalName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassifyBreak(t *testing.T) {
	accept := []string{
		"break",
		"modifier_break",
		"modifier_silver_edge_debuff",
		"modifier_vendetta_break",
		"modifier_angels_demise_break",
		"modifier_doom_bringer_doom_break",
		"modifier_viper_strike_debuff",
		"modifier_sharpshooter_debuff",
		"modifier_fan_of_knives",
		"modifier_berserker_troll_break",
		"MODIFIER_SILVER_EDGE_DEBUFF",
	}
	reject := []string{
		"modifier_break_stun",
		"modifier_breakfast",
		"modifier_doom_bringer_doom",
		"huskar_life_break_charge",
		"huskar_life_break_taunt",
		"modifier_puck_coil_break_stun",
		"item_mind_breaker",
		"special_bonus_mana_break",
		"spirit_breaker_greater_bash_break",
		"dawnbreaker_fire_wake",
		"modifier_silver_edge",
		"shadow_shaman_voodoo",
		"naga_siren_ensnare",
		"shadow_demon_purge_slow",
		"",
	}
	for _, inf := range accept {
		if !classifyBreak(inf) {
			t.Errorf("classifyBreak(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyBreak(inf) {
			t.Errorf("classifyBreak(%q) = true, want false", inf)
		}
	}
}

func TestClassifyFear(t *testing.T) {
	accept := []string{
		"fear",
		"modifier_fear",
		"modifier_terrorize",
		"modifier_sinister_gaze",
		"modifier_savage_roar",
		"modifier_dead_shot",
		"modifier_tame_the_beasts",
		"modifier_terror_wave",
		"modifier_aether_remnant",
		"modifier_will_o_wisp",
		"modifier_wheel_of_wonder",
	}
	reject := []string{
		"modifier_crippling_fear",
		"crippling_fear",
		"modifier_terror",
		"modifier_terrarium",
		"",
	}
	for _, inf := range accept {
		if !classifyFear(inf) {
			t.Errorf("classifyFear(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyFear(inf) {
			t.Errorf("classifyFear(%q) = true, want false", inf)
		}
	}
}

func TestClassifyRoot(t *testing.T) {
	accept := []string{
		"modifier_root",
		"modifier_frostbite",
		"modifier_searing_chains",
		"modifier_earthbind",
		"modifier_naga_siren_ensnare",
		"modifier_entangle",
		"modifier_oracle_fortunes_end_purge",
		"modifier_spinnerssnare",
		"modifier_rod_of_atos_debuff",
		"rod_of_atos",
	}
	reject := []string{
		"modifier_frost_bolt",
		"frost",
		"",
	}
	for _, inf := range accept {
		if !classifyRoot(inf) {
			t.Errorf("classifyRoot(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyRoot(inf) {
			t.Errorf("classifyRoot(%q) = true, want false", inf)
		}
	}
}

func TestClassifyLeash(t *testing.T) {
	accept := []string{
		"modifier_puck_coiled",
		"modifier_slark_pounce_leash",
		"modifier_soul_chain",
		"pounce_leash",
	}
	reject := []string{
		"modifier_puck_coil_break_stun",
		"modifier_pounce",
		"modifier_puck_whirling_death",
		"",
	}
	for _, inf := range accept {
		if !classifyLeash(inf) {
			t.Errorf("classifyLeash(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyLeash(inf) {
			t.Errorf("classifyLeash(%q) = true, want false", inf)
		}
	}
}

func TestClassifyTrap(t *testing.T) {
	accept := []string{
		"modifier_disruptor_kinetic_field",
		"modifier_mars_arena_of_blood_leash",
		"kinetic_field",
		"arena_of_blood_leash",
	}
	reject := []string{
		"modifier_mars_arena_of_blood_spear",
		"modifier_mars_arena_of_blood",
		"modifier_kinetic_energy",
		"modifier_kinetic",
		"",
	}
	for _, inf := range accept {
		if !classifyTrap(inf) {
			t.Errorf("classifyTrap(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyTrap(inf) {
			t.Errorf("classifyTrap(%q) = true, want false", inf)
		}
	}
}

func TestClassifyTaunt(t *testing.T) {
	accept := []string{
		"modifier_axe_berserkers_call",
		"berserkers_call",
		"berserkers_call_armor",
		"modifier_enchantress_little_friends_aura",
		"modifier_winter_wyvern_winters_curse_aura",
		"modifier_legion_commander_duel",
		"little_friends",
		"winters_curse",
		"modifier_huskar_life_break_taunt",
	}
	reject := []string{
		"modifier_huskar_life_break_charge",
		"modifier_huskar_life_break_slow_old",
		"modifier_life_break",
		"modifier_duel",
		"",
	}
	for _, inf := range accept {
		if !classifyTaunt(inf) {
			t.Errorf("classifyTaunt(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyTaunt(inf) {
			t.Errorf("classifyTaunt(%q) = true, want false", inf)
		}
	}
}

func TestClassifySilence(t *testing.T) {
	accept := []string{
		"modifier_silence",
		"modifier_doom_bringer_doom",
		"modifier_doom_bringer_doom_aura_enemy",
		"modifier_silencer_global_silence",
		"modifier_silencer_last_word",
		"modifier_night_stalker_crippling_fear",
		"modifier_static_storm",
		"modifier_ancient_seal",
		"modifier_hero_smoke_screen",
		"modifier_geomagnetic_grip_debuff",
		"modifier_orchid_malevolence",
		"modifier_bloodthorn_debuff",
		"modifier_ink_creature_debuff",
		"modifier_flux_alone",
		"modifier_ember_spirit_inner_fire_disarm",
	}
	reject := []string{
		"modifier_curse_of_the_silent",
		"modifier_silencer_last_word_disarm",
		"modifier_oppresive_pact",
		"modifier_primal_beast_oppresive_pact",
		"modifier_kunkka_torrent_knockback",
		"modifier_any_knockback",
		"modifier_knockback",
		"modifier_silencer",
		"modifier_ember_spirit_inner_fire",
		"modifier_oppresive_pact_but_silence",
		"",
	}
	for _, inf := range accept {
		if !classifySilence(inf) {
			t.Errorf("classifySilence(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifySilence(inf) {
			t.Errorf("classifySilence(%q) = true, want false", inf)
		}
	}
}

func TestClassifyDisarm(t *testing.T) {
	accept := []string{
		"modifier_disarmed",
		"modifier_disarm",
		"modifier_kunkka_disarmed",
		"modifier_invoker_deafening_blast_disarm",
		"modifier_techies_reactive_tazer_disarmed",
		"modifier_kobold_disarm",
		"modifier_roshan_revengeroar_disarm",
		"modifier_heavens_halberd_debuff",
		"modifier_sniper_concussive_grenade_slow",
		"modifier_oracle_fates_edict",
		"modifier_wave_blast_disarm",
		"modifier_shredder_chakram_disarm",
	}
	reject := []string{
		"modifier_lucky_shot",
		"modifier_pangolier_lucky_shot",
		"modifier_silencer_last_word",
		"modifier_silencer_last_word_disarm",
		"modifier_ember_spirit_inner_fire",
		"modifier_ember_spirit_inner_fire_disarm",
		"modifier_snapfire_scatterblast",
		"",
	}
	for _, inf := range accept {
		if !classifyDisarm(inf) {
			t.Errorf("classifyDisarm(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyDisarm(inf) {
			t.Errorf("classifyDisarm(%q) = true, want false", inf)
		}
	}
}

func TestClassifyHeal(t *testing.T) {
	accept := []string{
		"modifier_item_urn_heal",
		"modifier_item_spirit_vessel_heal",
		"modifier_item_essence_distiller_heal",
		"modifier_bottle_regeneration",
		"modifier_item_polliwog_charm_buff",
		"MODIFIER_ITEM_URN_HEAL",
	}
	reject := []string{
		"modifier_item_urn_heal_damage",
		"modifier_shadow_shaman_urnaconda",
		"modifier_item_urn_healing",
		"modifier_bottle_regeneration_other",
		"modifier_item_polliwog_charm",
		"modifier_item_pollywog_charm_buff",
		"modifier_item_spirit_vessel_healing",
		"urn_heal",
		"",
	}
	for _, inf := range accept {
		if !classifyHeal(inf) {
			t.Errorf("classifyHeal(%q) = false, want true", inf)
		}
	}
	for _, inf := range reject {
		if classifyHeal(inf) {
			t.Errorf("classifyHeal(%q) = true, want false", inf)
		}
	}
}

func TestLookupCategory(t *testing.T) {
	tests := []struct {
		inf  string
		want string
	}{
		{"modifier_dazzle_shallow_grave", "save"},
		{"eul_cyclone", "save"},
		{"modifier_item_aeon_disk", "save"},
		{"dazzle_shallow_grave", "save"},
		{"modifier_omniknight_repel", "spell_immunity"},
		{"modifier_ogre_magi_bloodlust", "buff_stats"},
		{"modifier_treant_living_armor", "buff_stats"},
		{"modifier_rune_invis", "invisibility"},
		{"modifier_item_guardian_greaves", "purge"},
		{"modifier_kotl_chakra_magic", "mana_restore"},
		{"modifier_dark_seer_surge", "buff_stats"},
		{"modifier_fountain_aura_buff", ""},
		{"modifier_any_unknown_thing", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := lookupCategory(tt.inf); got != tt.want {
			t.Errorf("lookupCategory(%q) = %q, want %q", tt.inf, got, tt.want)
		}
	}
}

func TestInstantHealItems(t *testing.T) {
	for _, item := range []string{"item_holy_locket", "item_famango", "item_great_famango", "item_greater_famango"} {
		if !InstantHealItems[item] {
			t.Errorf("InstantHealItems missing %q", item)
		}
	}
	for _, item := range []string{"item_urn_of_shadows", "item_dust", "modifier_item_holy_locket"} {
		if InstantHealItems[item] {
			t.Errorf("InstantHealItems should not contain %q", item)
		}
	}
}

func TestItemWardCosts(t *testing.T) {
	if ItemWardCosts["item_smoke_of_deceit"] != 50 {
		t.Error("smoke should cost 50")
	}
	if ItemWardCosts["item_dust"] != 80 {
		t.Error("dust should cost 80")
	}
}

func TestBuffCategoriesAliyOnly(t *testing.T) {
	for name, cat := range BuffCategories {
		if name == "" {
			t.Errorf("BuffCategories contains empty key")
		}
		switch cat {
		case "save", "purge", "shield", "spell_immunity", "invisibility", "buff_stats", "buff_haste", "mana_restore":
		default:
			t.Errorf("BuffCategories[%q] has unknown category %q", name, cat)
		}
	}
}
