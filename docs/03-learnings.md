# Glory — Captured Learnings

Cross-session insights from the planning sessions. Confidence 1-10.

## glory-thump-first-gate (architecture, confidence 8/10)

For Glory (WW1 terminal idle game in Go/Bubble Tea), the validation gate is the battle animation: build one satisfying offensive BEFORE the economy. The only stated differentiator is satisfying ASCII animation, so if the offensive does not grin, no curve tuning saves it.

## harmonica-spring-not-projectile (tool, confidence 8/10)

Charm Harmonica is a damped-spring solver, not a projectile solver. To animate a ballistic arc (e.g. artillery shells) drive x with a spring/lerp and y with a separate parabola; a lone spring on (x,y) yields a straight diagonal, not a curve.

## harmonica-juice-not-flight (tool, confidence 9/10)

In Glory, do NOT use Charm Harmonica for the shell/projectile flight (spring decelerates+settles+overshoots on BOTH axes; a shell is ballistic/monotonic). Shell = lerp(x)+parabola(y) over a timer, no physics lib. Reserve Harmonica for where settle/overshoot is the WANTED feel: impact recoil, screen-shake, resource bars springing to new values, number pop.

## glory-sim-anim-separation (architecture, confidence 9/10)

Glory: keep battle OUTCOME (who won, attrition) as a pure instantaneous ResolveBattle in internal/game; keep the charging/firing/resolving FSM + animation timing in internal/tui. v1 battles are manual (press F) so away-progress is accumulation-only and away-mode = one Tick(clampedDt). When auto-offensives/attrition land, away battles need a BOUNDED catch-up loop (resolve up to N capped), never one big-dt FSM call.

