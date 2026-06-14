package game

import "math"

// EffectType identifies which aspect of State an Upgrade modifies.
type EffectType int

const (
	// EffectMunitionsRate increases the passive munitions-per-second rate.
	EffectMunitionsRate EffectType = iota
	// EffectArmyPower increases the player's combat power.
	EffectArmyPower
)

// Upgrade describes a single purchasable upgrade in the WW1 economy.
// Adding a new upgrade is one data row in the Upgrades table.
type Upgrade struct {
	// ID is the unique machine-readable identifier used by Buy and OwnedCounts.
	ID string
	// Name is the human-readable display name.
	Name string
	// BaseCost is the cost when owned=0.
	BaseCost float64
	// CostGrowth is the exponential multiplier per owned copy.
	// Cost(owned) = BaseCost * CostGrowth^owned.
	CostGrowth float64
	// EffectType selects which stat the upgrade modifies.
	EffectType EffectType
	// EffectValue is the flat amount added to the selected stat per purchase.
	EffectValue float64
}

// Upgrades is the master table of purchasable upgrades.
// The slice is read-only at runtime; mutating it is a programming error.
var Upgrades = []Upgrade{
	{
		ID:          "supply_lines",
		Name:        "Supply Lines",
		BaseCost:    50.0,
		CostGrowth:  1.4,
		EffectType:  EffectMunitionsRate,
		EffectValue: 1.0, // +1 munition/sec
	},
	{
		ID:          "rifles",
		Name:        "Rifle Issue",
		BaseCost:    200.0,
		CostGrowth:  1.5,
		EffectType:  EffectArmyPower,
		EffectValue: 5.0, // +5 army power
	},
	{
		ID:          "artillery",
		Name:        "Artillery Battery",
		BaseCost:    500.0,
		CostGrowth:  1.6,
		EffectType:  EffectArmyPower,
		EffectValue: 20.0, // +20 army power
	},
}

// Cost returns the munitions price for the next copy of u when the player
// already owns owned copies.
//
//	Cost(u, owned) = u.BaseCost * u.CostGrowth^owned
func Cost(u Upgrade, owned int) float64 {
	return u.BaseCost * math.Pow(u.CostGrowth, float64(owned))
}

// Buy attempts to purchase one copy of the upgrade identified by id.
// It is a pure function: the input state is never mutated.
//
// If the upgrade is unknown or the player cannot afford it, Buy returns
// (s, false). Otherwise it returns a new State with the cost deducted, the
// owned count incremented, and the effect applied, plus (newState, true).
func Buy(s State, id string) (State, bool) {
	// Locate the upgrade.
	var found *Upgrade
	for i := range Upgrades {
		if Upgrades[i].ID == id {
			found = &Upgrades[i]
			break
		}
	}
	if found == nil {
		return s, false
	}

	owned := s.OwnedCounts[id]
	price := Cost(*found, owned)
	if s.Munitions < price {
		return s, false
	}

	// Build a new state (copy-on-write).
	next := s
	next.Munitions -= price

	// Copy the map so the original is not mutated.
	next.OwnedCounts = make(map[string]int, len(s.OwnedCounts)+1)
	for k, v := range s.OwnedCounts {
		next.OwnedCounts[k] = v
	}
	next.OwnedCounts[id] = owned + 1

	// Apply effect.
	switch found.EffectType {
	case EffectMunitionsRate:
		next.MunitionsRate += found.EffectValue
	case EffectArmyPower:
		next.ArmyPower += found.EffectValue
	}

	return next, true
}
