package game_test

import (
	"testing"

	"github.com/casualghoul/idle-glory/internal/game"
)

func TestResolveBattle_Tie_Stalemate(t *testing.T) {
	s := game.State{
		ArmyPower:    10.0,
		EnemyPower:   10.0,
		LinePosition: 0.0,
	}
	outcome := game.ResolveBattle(s)

	if outcome.Winner != game.WinnerStalemate {
		t.Errorf("equal power should be stalemate, got winner=%v", outcome.Winner)
	}
	if outcome.GroundGained != 0 {
		t.Errorf("stalemate must have no ground gained/lost, got GroundGained=%v", outcome.GroundGained)
	}
	if outcome.PlayerAttrition <= 0 {
		t.Errorf("stalemate should still have player attrition > 0, got %.4f", outcome.PlayerAttrition)
	}
	if outcome.EnemyAttrition <= 0 {
		t.Errorf("stalemate should still have enemy attrition > 0, got %.4f", outcome.EnemyAttrition)
	}
}

func TestResolveBattle_ClearWin(t *testing.T) {
	s := game.State{
		ArmyPower:    100.0,
		EnemyPower:   1.0,
		LinePosition: 0.0,
	}
	outcome := game.ResolveBattle(s)

	if outcome.Winner != game.WinnerPlayer {
		t.Errorf("player with 100x power should win, got winner=%v", outcome.Winner)
	}
	if outcome.GroundGained <= 0 {
		t.Errorf("player win should gain ground, got GroundGained=%v", outcome.GroundGained)
	}
	if outcome.PlayerAttrition < 0 {
		t.Errorf("player attrition should be >= 0, got %.4f", outcome.PlayerAttrition)
	}
	if outcome.EnemyAttrition <= 0 {
		t.Errorf("enemy attrition should be > 0 in player win, got %.4f", outcome.EnemyAttrition)
	}
}

func TestResolveBattle_ClearLoss(t *testing.T) {
	s := game.State{
		ArmyPower:    1.0,
		EnemyPower:   100.0,
		LinePosition: 0.0,
	}
	outcome := game.ResolveBattle(s)

	if outcome.Winner != game.WinnerEnemy {
		t.Errorf("enemy with 100x power should win, got winner=%v", outcome.Winner)
	}
	if outcome.GroundGained >= 0 {
		t.Errorf("enemy win should lose ground (negative), got GroundGained=%v", outcome.GroundGained)
	}
}

func TestResolveBattle_ZeroArmyPower(t *testing.T) {
	s := game.State{
		ArmyPower:    0.0,
		EnemyPower:   5.0,
		LinePosition: 0.0,
	}
	// Should not panic, enemy wins
	outcome := game.ResolveBattle(s)
	if outcome.Winner != game.WinnerEnemy {
		t.Errorf("zero army vs nonzero enemy should be enemy win, got %v", outcome.Winner)
	}
	if outcome.GroundGained >= 0 {
		t.Errorf("should lose ground with zero army, got GroundGained=%v", outcome.GroundGained)
	}
}

func TestResolveBattle_ZeroEnemyPower(t *testing.T) {
	s := game.State{
		ArmyPower:    5.0,
		EnemyPower:   0.0,
		LinePosition: 0.0,
	}
	outcome := game.ResolveBattle(s)
	if outcome.Winner != game.WinnerPlayer {
		t.Errorf("nonzero army vs zero enemy should be player win, got %v", outcome.Winner)
	}
	if outcome.GroundGained <= 0 {
		t.Errorf("should gain ground vs zero enemy, got GroundGained=%v", outcome.GroundGained)
	}
}

func TestResolveBattle_ZeroBothPowers(t *testing.T) {
	s := game.State{
		ArmyPower:  0.0,
		EnemyPower: 0.0,
	}
	// Both zero = tie = stalemate, should not panic
	outcome := game.ResolveBattle(s)
	if outcome.Winner != game.WinnerStalemate {
		t.Errorf("both-zero power should be stalemate, got %v", outcome.Winner)
	}
	if outcome.GroundGained != 0 {
		t.Errorf("stalemate must have GroundGained=0, got %v", outcome.GroundGained)
	}
}

func TestResolveBattle_IsPure(t *testing.T) {
	// Same inputs must give same outputs
	s := game.State{ArmyPower: 20.0, EnemyPower: 15.0}
	o1 := game.ResolveBattle(s)
	o2 := game.ResolveBattle(s)
	if o1 != o2 {
		t.Errorf("ResolveBattle must be pure: same input, different outputs: %+v vs %+v", o1, o2)
	}
}
