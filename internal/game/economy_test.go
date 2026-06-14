package game_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/andrewhorton/glory/internal/game"
)

func TestCost_CompoundingAcrossOwnedValues(t *testing.T) {
	u := game.Upgrade{
		ID:         "test",
		Name:       "Test Upgrade",
		BaseCost:   100.0,
		CostGrowth: 1.5,
	}

	cases := []struct {
		owned    int
		expected float64
	}{
		{0, 100.0},                      // 100 * 1.5^0 = 100
		{1, 150.0},                      // 100 * 1.5^1 = 150
		{2, 225.0},                      // 100 * 1.5^2 = 225
		{3, 337.5},                      // 100 * 1.5^3 = 337.5
		{10, 100.0 * math.Pow(1.5, 10)}, // 100 * 1.5^10
	}

	for _, tc := range cases {
		got := game.Cost(u, tc.owned)
		if math.Abs(got-tc.expected) > 1e-9 {
			t.Errorf("Cost(owned=%d): expected %.6f, got %.6f", tc.owned, tc.expected, got)
		}
	}
}

func TestBuy_Affordable(t *testing.T) {
	// Find artillery upgrade (or any upgrade with a known ID)
	var artilleryUpgrade *game.Upgrade
	for i := range game.Upgrades {
		if game.Upgrades[i].ID == "artillery" {
			artilleryUpgrade = &game.Upgrades[i]
			break
		}
	}
	if artilleryUpgrade == nil {
		t.Fatal("expected an upgrade with ID 'artillery' in Upgrades table")
	}

	cost := game.Cost(*artilleryUpgrade, 0)
	s := game.State{
		Munitions:   cost + 100.0, // more than enough
		OwnedCounts: map[string]int{},
	}

	newState, ok := game.Buy(s, "artillery")
	if !ok {
		t.Fatal("Buy should return true when affordable")
	}
	if newState.Munitions >= s.Munitions {
		t.Errorf("Munitions should decrease after purchase; before=%.2f, after=%.2f",
			s.Munitions, newState.Munitions)
	}
	if newState.OwnedCounts["artillery"] != s.OwnedCounts["artillery"]+1 {
		t.Errorf("owned count should increment by 1")
	}
	// Original state must be unchanged (pure function)
	if s.OwnedCounts["artillery"] != 0 {
		t.Errorf("Buy must not mutate input state")
	}
}

func TestBuy_NotAffordable(t *testing.T) {
	var artilleryUpgrade *game.Upgrade
	for i := range game.Upgrades {
		if game.Upgrades[i].ID == "artillery" {
			artilleryUpgrade = &game.Upgrades[i]
			break
		}
	}
	if artilleryUpgrade == nil {
		t.Fatal("expected an upgrade with ID 'artillery' in Upgrades table")
	}

	cost := game.Cost(*artilleryUpgrade, 0)
	s := game.State{
		Munitions:   cost - 1.0, // one short
		OwnedCounts: map[string]int{},
	}

	newState, ok := game.Buy(s, "artillery")
	if ok {
		t.Fatal("Buy should return false when not affordable")
	}
	if !reflect.DeepEqual(newState, s) {
		t.Errorf("Buy should return original state unchanged when not affordable")
	}
}

func TestBuy_UnknownID(t *testing.T) {
	s := game.State{
		Munitions:   999999.0,
		OwnedCounts: map[string]int{},
	}
	_, ok := game.Buy(s, "nonexistent_upgrade")
	if ok {
		t.Error("Buy with unknown ID should return false")
	}
}

func TestBuy_EffectIncreasesRate(t *testing.T) {
	// At least one upgrade should increase MunitionsRate (passive munitions/sec).
	// Find an upgrade with EffectType that raises passive rate.
	var rateUpgrade *game.Upgrade
	for i := range game.Upgrades {
		if game.Upgrades[i].EffectType == game.EffectMunitionsRate {
			rateUpgrade = &game.Upgrades[i]
			break
		}
	}
	if rateUpgrade == nil {
		t.Fatal("expected at least one upgrade with EffectType=EffectMunitionsRate in Upgrades table")
	}

	cost := game.Cost(*rateUpgrade, 0)
	s := game.State{
		Munitions:     cost + 100.0,
		MunitionsRate: 1.0,
		OwnedCounts:   map[string]int{},
	}

	newState, ok := game.Buy(s, rateUpgrade.ID)
	if !ok {
		t.Fatal("Buy should succeed")
	}
	if newState.MunitionsRate <= s.MunitionsRate {
		t.Errorf("MunitionsRate should increase after buying a rate upgrade; before=%.2f, after=%.2f",
			s.MunitionsRate, newState.MunitionsRate)
	}
}

func TestBuy_EffectIncreasesArmyPower(t *testing.T) {
	// Find an upgrade that increases army power.
	var powerUpgrade *game.Upgrade
	for i := range game.Upgrades {
		if game.Upgrades[i].EffectType == game.EffectArmyPower {
			powerUpgrade = &game.Upgrades[i]
			break
		}
	}
	if powerUpgrade == nil {
		t.Fatal("expected at least one upgrade with EffectType=EffectArmyPower")
	}

	cost := game.Cost(*powerUpgrade, 0)
	s := game.State{
		Munitions:   cost + 100.0,
		ArmyPower:   5.0,
		OwnedCounts: map[string]int{},
	}

	newState, ok := game.Buy(s, powerUpgrade.ID)
	if !ok {
		t.Fatal("Buy should succeed")
	}
	if newState.ArmyPower <= s.ArmyPower {
		t.Errorf("ArmyPower should increase; before=%.2f, after=%.2f",
			s.ArmyPower, newState.ArmyPower)
	}
}

func TestUpgrades_TableHasRequiredUpgrades(t *testing.T) {
	ids := make(map[string]bool)
	for _, u := range game.Upgrades {
		ids[u.ID] = true
	}
	required := []string{"artillery", "rifles", "supply_lines"}
	for _, id := range required {
		if !ids[id] {
			t.Errorf("expected upgrade with ID %q in Upgrades table", id)
		}
	}
	if len(game.Upgrades) < 3 {
		t.Errorf("expected at least 3 upgrades, got %d", len(game.Upgrades))
	}
}
