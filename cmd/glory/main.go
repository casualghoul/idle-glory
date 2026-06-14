// Command glory is the entrypoint for Glory, a WW1 terminal idle game.
//
// Run with -headless to execute a deterministic core-loop smoke test suitable
// for CI. Without -headless, the command prints the current state summary and
// exits (interactive TUI is a later task).
//
// Environment variables:
//
//	GLORY_SAVE_DIR   override for the save directory (default: XDG config path)
//	GLORY_HEADLESS   set to "1" to enable headless mode without the flag
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/andrewhorton/glory/internal/game"
	"github.com/andrewhorton/glory/internal/save"
)

// headlessConfig holds the parameters for a headless run.
// Keeping them in a struct makes runHeadless unit-testable with any inputs.
type headlessConfig struct {
	// ticks is the number of simulation ticks to advance.
	ticks int
	// tickDt is the duration of each tick.
	tickDt time.Duration
}

// defaultHeadlessConfig is the configuration used by the -headless flag.
var defaultHeadlessConfig = headlessConfig{
	ticks:  30,
	tickDt: time.Second,
}

// newState returns the canonical starting state for a fresh game.
//
// Rationale for placement: internal/game is frozen after review and is a pure
// sim package with no concept of "game start" vs "any valid state". The
// starting configuration (initial rates, enemy setup) is a product decision
// that belongs at the entrypoint layer, not the simulation layer.
func newState() game.State {
	return game.State{
		Munitions:     0,
		MunitionsRate: 1.0,  // 1 munition/sec so accumulation is observable immediately
		ArmyPower:     10.0, // starting force
		EnemyPower:    10.0, // equal opponent — stalemate until player upgrades
		LinePosition:  0,
		OwnedCounts:   make(map[string]int),
	}
}

// runHeadless advances state by cfg.ticks * cfg.tickDt, buying the cheapest
// affordable MunitionsRate upgrade after each tick if one is available.
// It is a pure function of its inputs and produces no I/O.
func runHeadless(state game.State, _ save.Clock, cfg headlessConfig) game.State {
	for i := 0; i < cfg.ticks; i++ {
		state = game.Tick(state, cfg.tickDt)
		// Attempt to buy the cheapest affordable MunitionsRate upgrade.
		for _, u := range game.Upgrades {
			if u.EffectType == game.EffectMunitionsRate {
				if next, ok := game.Buy(state, u.ID); ok {
					state = next
					break
				}
			}
		}
	}
	return state
}

// printState writes a human-readable summary of s to w.
func printState(w *os.File, s game.State) {
	fmt.Fprintf(w, "  Munitions:     %s\n", game.FormatNum(s.Munitions))
	fmt.Fprintf(w, "  Rate:          %s/s\n", game.FormatNum(s.MunitionsRate))
	fmt.Fprintf(w, "  Army Power:    %s\n", game.FormatNum(s.ArmyPower))
	fmt.Fprintf(w, "  Enemy Power:   %s\n", game.FormatNum(s.EnemyPower))
	fmt.Fprintf(w, "  Line Position: %s\n", game.FormatNum(s.LinePosition))
	fmt.Fprintf(w, "  Upgrades:\n")
	for _, u := range game.Upgrades {
		owned := s.OwnedCounts[u.ID]
		nextCost := game.Cost(u, owned)
		fmt.Fprintf(w, "    %-20s owned=%d  next=%s\n",
			u.Name, owned, game.FormatNum(nextCost))
	}
}

func main() {
	headlessFlag := flag.Bool("headless", false,
		"run a deterministic core-loop demo, save, and exit (useful for CI)")
	flag.Parse()

	headless := *headlessFlag || os.Getenv("GLORY_HEADLESS") == "1"

	// Resolve save directory — prefer explicit env override, then XDG default.
	saveDir := os.Getenv("GLORY_SAVE_DIR")
	if saveDir == "" {
		var err error
		saveDir, err = save.XDGConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glory: resolve save dir: %v\n", err)
			os.Exit(1)
		}
	}

	clk := save.SystemClock{}

	// Load existing save (or start fresh).
	out, err := save.Load(saveDir, clk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glory: load save: %v\n", err)
		os.Exit(1)
	}

	var state game.State

	switch out.Result {
	case save.LoadMissing:
		fmt.Println("No save found — starting a new game.")
		state = newState()

	case save.LoadCorrupt:
		// Never silently swallow a corrupt-save message.
		fmt.Printf("Warning: %s\n", out.Message)
		fmt.Println("Starting a new game.")
		state = newState()

	case save.LoadOK:
		away := save.ApplyAwayProgress(out.State, out.SavedAt, clk)
		state = away.State
		if away.Duration > 0 {
			fmt.Printf("While you were away (%v):\n", away.Duration.Round(time.Second))
			fmt.Printf("  Munitions gained: %s\n", game.FormatNum(away.MunitionsGained))
		}
	}

	fmt.Println("\nCurrent state:")
	printState(os.Stdout, state)

	if headless {
		fmt.Println("\n[Headless] Running core-loop demo...")
		state = runHeadless(state, clk, defaultHeadlessConfig)
		fmt.Println("\nAfter demo ticks:")
		printState(os.Stdout, state)

		if err := save.Save(saveDir, clk, state); err != nil {
			fmt.Fprintf(os.Stderr, "glory: save: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nSaved to %s\n", save.SavePath(saveDir))
		return
	}

	// TODO(T4): launch Bubble Tea program here.
	fmt.Println("\nInteractive TUI not yet implemented — run with -headless for the core-loop demo.")

	// Save on exit so the next launch can apply away-progress.
	if err := save.Save(saveDir, clk, state); err != nil {
		fmt.Fprintf(os.Stderr, "glory: save: %v\n", err)
		os.Exit(1)
	}
}
