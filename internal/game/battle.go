package game

// Winner identifies the result of a battle.
type Winner int

const (
	// WinnerStalemate means both sides had equal power: no ground changed.
	WinnerStalemate Winner = iota
	// WinnerPlayer means the player's army had superior power.
	WinnerPlayer
	// WinnerEnemy means the enemy had superior power.
	WinnerEnemy
)

// Outcome is the instantaneous, pure result of a single battle evaluation.
type Outcome struct {
	// Winner is the result: WinnerStalemate, WinnerPlayer, or WinnerEnemy.
	Winner Winner
	// GroundGained is the change to LinePosition (positive = player advances,
	// negative = player retreats). Zero on stalemate.
	GroundGained float64
	// PlayerAttrition is the army-power loss dealt to the player.
	PlayerAttrition float64
	// EnemyAttrition is the army-power loss dealt to the enemy.
	EnemyAttrition float64
}

// attritionFraction is the fraction of the losing side's power the winner
// deals as casualties. Both sides always take some attrition even on stalemate.
const attritionFraction = 0.1

// groundPerPowerRatio converts power difference to ground gained/lost per battle.
const groundPerPowerRatio = 0.05

// ResolveBattle is a pure, instantaneous evaluation of the player's army
// against the current enemy. It never loops, animates, or reads global state.
//
//   - If ArmyPower > EnemyPower  → WinnerPlayer, positive GroundGained.
//   - If ArmyPower < EnemyPower  → WinnerEnemy,  negative GroundGained.
//   - If ArmyPower == EnemyPower → WinnerStalemate, GroundGained = 0;
//     both sides still take attrition casualties.
func ResolveBattle(s State) Outcome {
	p := s.ArmyPower
	e := s.EnemyPower

	switch {
	case p > e:
		return Outcome{
			Winner:          WinnerPlayer,
			GroundGained:    (p - e) * groundPerPowerRatio,
			PlayerAttrition: e * attritionFraction,
			EnemyAttrition:  p * attritionFraction,
		}
	case e > p:
		return Outcome{
			Winner:          WinnerEnemy,
			GroundGained:    -(e - p) * groundPerPowerRatio,
			PlayerAttrition: e * attritionFraction,
			EnemyAttrition:  p * attritionFraction,
		}
	default: // tie — includes both-zero
		return Outcome{
			Winner:          WinnerStalemate,
			GroundGained:    0,
			PlayerAttrition: e * attritionFraction,
			EnemyAttrition:  p * attritionFraction,
		}
	}
}
