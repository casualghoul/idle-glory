# Glory — Planning Docs

A WW1-themed terminal **idle game** in Go, designed to be played in the downtime
while AI coding agents run. Live TUI (Bubble Tea), satisfying ASCII animation.

These docs are the output of two planning passes (`/office-hours` product design,
then `/plan-eng-review` architecture lock). They are the source of truth for
implementation.

## Read in this order

1. **[`01-design.md`](01-design.md)** — product design: what the game is, the core
   loop (accumulation + battles + prestige + attrition), the "reward the glance"
   philosophy, premises, and milestones. The *what* and *why*.
2. **[`02-engineering-spec.md`](02-engineering-spec.md)** — locked architecture: package
   layout, all engineering decisions (D1–D11), failure modes, testing, and
   parallelization. The *how*. **Where it conflicts with the design doc, this wins**
   (it includes a review pass that revised two decisions).
3. **[`03-learnings.md`](03-learnings.md)** — cross-session gotchas worth knowing before
   you start (e.g. Harmonica is a spring not a projectile solver).
4. **[`../TODOS.md`](../TODOS.md)** — deferred work with full context (the away-battle
   catch-up loop).
5. **[`tasks.jsonl`](tasks.jsonl)** — machine-readable task list (T1–T8) with priority,
   component, effort, files, and the finding each derives from.

## Build order (start here)

1. **T1 — headless core-loop spike** (no animation): `Tick` + accumulate + save +
   quit + relaunch-with-away-progress. Confirm numbers + clamp behave.
2. **T5 — thump spike** (the reward): one offensive that arcs and resolves. Make it grin.
3. **T2/T3/T4/T6 — v1 vertical slice**: pure `internal/game`, durable `internal/save`,
   the live `internal/tui`, full tests alongside.
4. **T7 — distribution**: GoReleaser + GitHub Actions.
5. **Layer phase** (post-slice, see `01-design.md`): prestige, attrition, multiple
   fronts, terminal-activity sync.

## The one rule that matters most

`internal/game` is a **pure simulation** that imports zero TUI/animation code. It is
unit-tested with no terminal. Battle *outcomes* are pure; battle *animation/timing*
lives in `internal/tui`. Keep this boundary rigid — every good property downstream
(deterministic tests, the headless spike, correct away-progress) depends on it.
