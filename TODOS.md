# TODOS

## Bounded away-battle catch-up loop
- **What:** When away-progress must resolve battles (not just accumulate resources), implement a bounded catch-up loop that resolves up to N battles with a step cap — never a single big-dt call into the battle resolver.
- **Why:** v1 battles are manual (press F), so away-progress is accumulation-only and `Tick(clampedDt)` is correct. Once auto-offensives / attrition land, battles can occur while the player is away. Feeding the resolver one multi-hour dt would either fast-forward thousands of battles in one step or silently swallow them. A capped loop resolves a defined, testable number.
- **Pros:** Keeps away-mode battle outcomes correct and deterministic; preserves the strict sim/animation split (outcome math is pure; the FSM/animation is TUI-only).
- **Cons:** Needs a sensible cap policy (max battles per catch-up, or max simulated time) and tests for the cap boundary.
- **Context:** Surfaced in the /plan-eng-review outside-voice pass (2026-06-14). Depends on the pure, instantaneous `ResolveBattle` in `internal/game` (decision T2-A) and on the auto-offensive/attrition layer existing. The pure resolver is the prerequisite — build it instantaneous from day one so this loop is just "call it N times under a cap."
- **Depends on / blocked by:** Auto-offensive / attrition layer (post-v1-slice); pure instantaneous `ResolveBattle`.
