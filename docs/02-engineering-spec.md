# Glory — Engineering Spec (locked architecture)

Source: `/plan-eng-review` session, 2026-06-14. This is the authoritative
architecture for implementation. Pairs with `01-design.md` (the product design).
Where the design doc and this spec disagree, **this spec wins** — it incorporates
an outside-voice review pass that revised two earlier decisions (noted below).

## TL;DR build order (revised)

1. **Headless core-loop spike FIRST** (no animation). `Tick` + accumulate + save +
   quit + relaunch-with-away-progress. Run it over a coffee break, confirm the
   numbers and the clock-skew clamp behave. This retires the real architectural
   risk. *(Reverses the design doc's "thump-first" — see Decision T1.)*
2. **Thump spike SECOND** (the reward): one offensive that arcs and resolves on
   screen, make it grin.
3. **v1 vertical slice:** wire the above into the live TUI with the table-driven
   economy, upgrades, and full save durability.
4. **Layer phase:** prestige eras, attrition, multiple fronts, terminal-activity
   sync. (Out of scope here; see `01-design.md`.)

## Package layout (strict boundary — the load-bearing decision)

```
cmd/glory/            main(); wires everything, owns the Bubble Tea program
internal/game/        PURE simulation. Imports ZERO tui/charm packages.
                      Tick, Cost, Buy, ResolveBattle, FormatNum, types, []Upgrade
internal/tui/         Bubble Tea model/update/view; the battle FSM; animation
internal/tui/anim/    shell flight + Harmonica juice
internal/save/        JSON persistence, Clock interface, atomic write, away-progress
```

**Rule (keep it rigid):** `internal/game` is import-clean — no terminal, no Charm,
no animation. It is unit-tested with no TTY. Animation *timing* is presentation and
lives in `internal/tui`, never in the sim. This is the spine; everything good
(deterministic tests, the headless spike, away-mode) depends on it.

## Locked decisions

### Architecture
- **D1 — Layered packages, strict pure-core boundary.** As above. `internal/game`
  imports nothing from `internal/tui`.
- **D2 — Single dt-based clock; fixed ~30fps interval to start.** One `tickMsg`
  carries a wall-clock timestamp. Each tick: `dt = now - lastTick`; advance the sim
  by `dt`; accumulation `= rate * dt` (so tick frequency never affects totals).
  Schedule the next `tea.Tick` at a **fixed ~30ms** interval for now. *(Revised from
  an adaptive clock — see T3. Adaptive idle-slowdown is a one-line change later;
  defer until a profiler asks.)*
- **D3 — Clock interface + away-progress = `Tick(bigDt)`.** Pure funcs take only
  `dt time.Duration` (no clock). A small `Clock` interface (`Now() time.Time`) is
  injected at the save boundary. On launch:
  `dt = clamp(now - lastSaved, 0, cap)`, then call the **same** `Tick` used live.
  One accumulation path for live + catch-up. A fake clock makes negative-dt and
  huge-dt cases trivially unit-testable.
- **D4 — Numbers are plain `float64` + one `FormatNum()`.** No `type Num` alias
  (an alias gives zero swap protection — it was cargo-cult). `FormatNum` is the one
  real abstraction (K / M / B / scientific). float64 is exact to ~9e15; numbers are
  display-formatted anyway. Revisit a bignum type only if exactness past ~1e15 ever
  matters. *(Revised — see T4.)*

### Game logic
- **D5 — Table-driven economy from v1.** `[]Upgrade{ID, Name, BaseCost, CostGrowth,
  Effect}` + a generic `Buy(state, id)`. `Cost(u, owned) = u.BaseCost *
  u.CostGrowth^owned`. Adding an upgrade = one data row (+ maybe a small effect
  closure). This is the in-code bridge to the design's eventual external config —
  no logic change when it later loads from embedded TOML/JSON.
- **D6 — `ResolveBattle` is PURE and INSTANTANEOUS.** It returns the outcome (who
  won + attrition dealt) from a single evaluation of `yourPower` vs `theirPower`.
  The charging → firing → resolving **FSM and its timing live in `internal/tui`** as
  presentation wrapping the instantaneous result.
- **D7 — v1 battles are MANUAL (press `F`).** Therefore away-progress is
  accumulation-only and D3's "one Tick = live + catch-up" is literally true. When
  auto-offensives/attrition arrive (layer phase), away-mode needs a **bounded
  catch-up loop** (resolve up to N battles, capped) — captured in `TODOS.md`, NOT
  built now.
- **D8 — Battle tie = stalemate.** Exact power tie → no ground gained or lost, both
  sides take attrition casualties. Thematically WW1, deterministic, testable.

### Animation (D9)
- **Shell flight = `lerp(x)` + `parabola(y)` over a fixed timer. No physics lib.**
  Harmonica is a *spring* solver (decelerates, settles, overshoots) and is wrong for
  a ballistic shell on **both** axes.
- **Harmonica earns its place on the JUICE:** impact recoil, screen-shake, resource
  bars springing to new values, number pop — where settle/overshoot is the *wanted*
  feel.
- **Screen-shake** = offset the entire rendered view by ±N columns for a few frames.

### Persistence durability (D10)
- **Atomic write:** write `save.json.tmp`, then `os.Rename` (atomic on same FS). A
  crash never leaves a half-written save.
- **Corrupt/failed load:** back up the bad file to `save.json.corrupt-<ts>`, start a
  fresh game, surface a message. **NEVER silent-wipe.**
- **Version guard:** save carries a `version` field. Older → migrate via additive
  defaults. Newer (downgrade) → back up + fresh, do not crash.
- **Away-progress delta guard:** clamp `now - lastSaved` to `[0, cap]` — a backward
  clock can't produce negative/garbage earnings; a 3-week sleep can't overflow into
  an instant win.
- **Save cadence:** on quit (**catch SIGTERM / ctrl-c**), on purchase, and periodic
  autosave (so a laptop-close mid-session loses at most one interval).
- **Paths (XDG):** config under `~/.config/glory/`, save at
  `~/.config/glory/save.json` (or `~/.local/state/glory/`).

## Testing (D11 — full pyramid)
- `internal/game` + `internal/save` → ~100% via table tests, including: dt=0,
  dt huge (post-clamp), tie, zero-army, cost compounding, FormatNum boundaries,
  corrupt load, negative-dt clamp, huge-dt clamp, version older/newer.
- Battle FSM → deterministic unit tests: drive with fixed dt steps, assert each
  transition. **Do NOT golden-snapshot mid-animation frames (timing-flaky).**
- TUI → `teatest` golden files on **static** frames only (idle / post-buy /
  post-resolve).
- **quit-saves-progress integration test** — the data-loss path; highest value.

## Failure modes (all covered unless noted)
| Failure | Mitigation | Note |
|---|---|---|
| Save interrupted mid-write | atomic tmp+rename | old save intact |
| Corrupt/truncated save | backup + fresh + message | never silent-wipe |
| Clock backward / long sleep | clamp `[0, cap]` | correct catch-up |
| Newer save version | backup + fresh, no crash | |
| **Terminal resize** | **handle `WindowSizeMsg` in `internal/tui`** | **MUST implement — garbles otherwise** |
| SIGTERM / ctrl-c / laptop close | save-on-quit + periodic autosave | bounded loss worst case |

Also handle: alt-screen mode (`tea.WithAltScreen`) so the game doesn't pollute
scrollback; focus blur/regain is optional polish.

## Parallelization (for multi-agent implementation)
| Lane | Modules | Depends on |
|------|---------|-----------|
| A | `internal/game` (pure sim + table economy) | — |
| B | `internal/save` (Clock iface, atomic write) | game types (light) |
| C | `internal/tui` + `internal/tui/anim` | A + B |

Lane A and Lane B's interface/atomic-write scaffolding can run in parallel
worktrees; merge, then build Lane C on top. Low conflict risk (separate packages).

## Dependencies
- `github.com/charmbracelet/bubbletea` — loop + `tea.Tick`
- `github.com/charmbracelet/lipgloss` — styled ASCII layout
- `github.com/charmbracelet/harmonica` — spring physics (JUICE only, per D9)
- `github.com/charmbracelet/x/exp/teatest` — TUI golden tests
- `goreleaser` + GitHub Actions — release (T7)

## Implementation tasks
See `tasks.jsonl` (T1–T8) and the "Implementation Tasks" list. Priorities: T1–T6
are P1 (the v1 slice), T7 is P2 (distribution), T8 is P3/deferred (TODOS.md).
