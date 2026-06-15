package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/casualghoul/idle-glory/internal/game"
	"github.com/casualghoul/idle-glory/internal/tui/anim"
)

// 16-color-safe palette (ANSI). Using named ANSI indices keeps the game
// legible on basic terminals and over SSH.
var (
	colMud       = lipgloss.Color("3")  // dark yellow / khaki
	colSky       = lipgloss.Color("6")  // cyan-ish sky
	colPlayer    = lipgloss.Color("2")  // green — our line
	colEnemy     = lipgloss.Color("1")  // red — the enemy
	colShell     = lipgloss.Color("11") // bright yellow — the shell + flash
	colMuted     = lipgloss.Color("8")  // grey — secondary text
	colHi        = lipgloss.Color("15") // bright white — headlines
	colVictory   = lipgloss.Color("10") // bright green
	colDefeat    = lipgloss.Color("9")  // bright red
	colStalemate = lipgloss.Color("7")  // white
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(colHi)
	mutedStyle  = lipgloss.NewStyle().Foreground(colMuted)
	rateStyle   = lipgloss.NewStyle().Foreground(colSky)
	munStyle    = lipgloss.NewStyle().Bold(true).Foreground(colShell)
	playerStyle = lipgloss.NewStyle().Foreground(colPlayer)
	enemyStyle  = lipgloss.NewStyle().Foreground(colEnemy)
	shellStyle  = lipgloss.NewStyle().Bold(true).Foreground(colShell)
	paneStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colMud).
			Padding(0, 1)
)

// noMansLandDims computes the inner grid (cols, rows) for the middle pane given
// the terminal size. Kept as a free function so update.go and view.go agree.
func noMansLandDims(width, height int) (cols, rows int) {
	// Account for borders + padding (~4 cols) and the other two panes + chrome.
	cols = width - 6
	if cols < 10 {
		cols = 10
	}
	rows = height - 18 // top pane (~6) + bottom pane (~4) + borders/banner
	if rows < 5 {
		rows = 5
	}
	if rows > 12 {
		rows = 12 // a tall arc is enough; don't waste vertical space
	}
	return cols, rows
}

// View renders the three stacked panes and applies the screen-shake offset.
func (m Model) View() string {
	if m.quitting {
		return "Holding the line. Saving… see you in the trenches.\n"
	}

	width := m.width
	if width < 1 {
		width = 80
	}

	var b strings.Builder
	if m.showAway && m.awayBanner.Duration > 0 {
		b.WriteString(m.renderAwayBanner(width))
		b.WriteString("\n")
	}
	b.WriteString(m.renderTop(width))
	b.WriteString("\n")
	b.WriteString(m.renderMiddle(width))
	b.WriteString("\n")
	b.WriteString(m.renderBottom(width))

	out := b.String()

	// Screen-shake: pad the whole view left by N columns for a few frames.
	if off := m.shake.Offset(); off != 0 {
		out = shiftColumns(out, off)
	}
	return out
}

// shiftColumns offsets every line of s right by abs(n) spaces (the shake reads
// as a recoil jolt). Negative n also shifts right — direction is cosmetic.
// Lines contain zero-width ANSI escape codes, so prepending ASCII spaces
// produces the correct visual column shift for screen-shake without disturbing
// the rendered glyphs.
func shiftColumns(s string, n int) string {
	if n < 0 {
		n = -n
	}
	if n == 0 {
		return s
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

// renderAwayBanner shows the "while you were away" summary on first render.
func (m Model) renderAwayBanner(width int) string {
	msg := fmt.Sprintf("While you were away (%s): +%s munitions",
		m.awayBanner.Duration.Round(time.Second), game.FormatNum(m.awayBanner.MunitionsGained))
	style := lipgloss.NewStyle().
		Foreground(colHi).
		Background(colMud).
		Bold(true).
		Padding(0, 1).
		Width(min(width, lipgloss.Width(msg)+4))
	return style.Render(msg)
}

// renderTop is the economy + upgrades pane.
func (m Model) renderTop(width int) string {
	displayMun := m.munitionsSpring.Value()
	if displayMun < 0 {
		displayMun = 0
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		titleStyle.Render("⚙ MUNITIONS "),
		munStyle.Render(game.FormatNum(displayMun)),
		rateStyle.Render(fmt.Sprintf("  +%s/s", game.FormatNum(m.state.MunitionsRate))),
	)

	var rows []string
	rows = append(rows, header)
	rows = append(rows, mutedStyle.Render("─── upgrades ───"))
	for i, u := range game.Upgrades {
		owned := m.state.OwnedCounts[u.ID]
		cost := game.Cost(u, owned)
		affordable := m.state.Munitions >= cost
		keyLabel := fmt.Sprintf("[%d]", i+1)

		line := fmt.Sprintf("%s %-18s owned %-3d  cost %s",
			keyLabel, u.Name, owned, game.FormatNum(cost))
		st := mutedStyle
		if affordable {
			st = lipgloss.NewStyle().Foreground(colPlayer)
		}
		rows = append(rows, st.Render(line))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return paneStyle.Width(paneInnerWidth(width)).Render(body)
}

// renderMiddle is the no-man's-land — the money shot.
func (m Model) renderMiddle(width int) string {
	cols, rows := noMansLandDims(m.width, m.height)
	grid := anim.NewRuneGrid(rows, cols, ' ')

	groundRow := rows - 1

	// Our trench on the far left, enemy trench on the far right.
	plotString(grid, 0, groundRow, "|‾|")
	enemyTrenchCol := cols - 3
	plotString(grid, enemyTrenchCol, groundRow, "|‾|")

	// Dotted no-man's-land along the ground between the trenches.
	for c := 3; c < enemyTrenchCol; c++ {
		anim.PlotShell(grid, c, groundRow, '.')
	}

	// Enemy front line marker — recedes (moves right, toward their trench) as
	// the player gains ground. LinePosition>0 = player advance.
	linePos := m.lineSpring.Value()
	// Map line position to a column: center, shifted by ground gained.
	mid := cols / 2
	lineCol := mid + int(linePos)
	if lineCol < 3 {
		lineCol = 3
	}
	if lineCol > enemyTrenchCol-1 {
		lineCol = enemyTrenchCol - 1
	}
	anim.PlotShell(grid, lineCol, groundRow, '#')

	// Shell in flight.
	if m.battle.shellLive {
		anim.PlotShell(grid, int(m.battle.shellX), int(m.battle.shellY), '◉')
	}

	// Impact flash: replace ground around the impact with a burst for the first
	// part of the resolve window.
	if m.battle.phase == phaseResolving && m.battle.resolveFraction() < 0.5 {
		burstCol := int(m.battle.flight.EndX)
		for d := -2; d <= 2; d++ {
			anim.PlotShell(grid, burstCol+d, groundRow, '*')
		}
		anim.PlotShell(grid, burstCol, groundRow-1, '*')
	}

	body := renderGrid(grid, m.battle)

	// Outcome banner during resolve.
	var status string
	switch {
	case m.battle.phase == phaseResolving:
		status = renderOutcome(m.battle.outcome)
	case m.battle.shellLive:
		status = shellStyle.Render("Shell away — incoming!")
	default:
		status = mutedStyle.Render(fmt.Sprintf("Line: %s   |   our trench ⟵   ⟶ enemy trench",
			game.FormatNum(m.state.LinePosition)))
	}

	full := lipgloss.JoinVertical(lipgloss.Left, body, status)
	return paneStyle.Width(paneInnerWidth(width)).Render(full)
}

// renderBottom is the fire control + charge meter + legend.
func (m Model) renderBottom(width int) string {
	frac := m.battle.chargeFraction()
	barWidth := width / 3
	if barWidth < 8 {
		barWidth = 8
	}
	bar := anim.Bar(frac, 1.0, barWidth)

	var fireLabel string
	switch m.battle.phase {
	case phaseFiring:
		fireLabel = shellStyle.Render("[F] FIRING…")
	case phaseResolving:
		fireLabel = mutedStyle.Render("[F] Impact!")
	default:
		if m.battle.ready() {
			fireLabel = lipgloss.NewStyle().Bold(true).Foreground(colVictory).Render("[F] FIRE OFFENSIVE — READY")
		} else {
			fireLabel = mutedStyle.Render("[F] Fire Offensive (charging)")
		}
	}

	barStyle := mutedStyle
	if m.battle.ready() {
		barStyle = lipgloss.NewStyle().Foreground(colShell)
	}

	meter := lipgloss.JoinHorizontal(lipgloss.Left,
		fireLabel,
		"  ",
		barStyle.Render(bar),
	)

	legend := mutedStyle.Render("F/space fire   1-3 buy upgrade   q quit")

	rowsToJoin := []string{meter, legend}
	if m.saveErr != nil {
		rowsToJoin = append(rowsToJoin,
			lipgloss.NewStyle().Foreground(colDefeat).Render("save error: "+m.saveErr.Error()))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rowsToJoin...)
	return paneStyle.Width(paneInnerWidth(width)).Render(body)
}

// renderOutcome styles the battle result line.
func renderOutcome(o game.Outcome) string {
	var head string
	var st lipgloss.Style
	switch o.Winner {
	case game.WinnerPlayer:
		head = "VICTORY — ground gained!"
		st = lipgloss.NewStyle().Bold(true).Foreground(colVictory)
	case game.WinnerEnemy:
		head = "REPULSED — we lost ground."
		st = lipgloss.NewStyle().Bold(true).Foreground(colDefeat)
	default:
		head = "STALEMATE — the line holds."
		st = lipgloss.NewStyle().Bold(true).Foreground(colStalemate)
	}
	detail := fmt.Sprintf("  Δline %+.2f   our losses -%s   enemy losses -%s",
		o.GroundGained, game.FormatNum(o.PlayerAttrition), game.FormatNum(o.EnemyAttrition))
	return st.Render(head) + mutedStyle.Render(detail)
}

// renderGrid converts a RuneGrid to a styled multi-line string. The shell and
// burst glyphs are colorised; everything else is mud/sky toned.
func renderGrid(g anim.RuneGrid, b battle) string {
	var lines []string
	for _, row := range g {
		var sb strings.Builder
		for _, r := range row {
			switch r {
			case '◉':
				sb.WriteString(shellStyle.Render(string(r)))
			case '*':
				sb.WriteString(lipgloss.NewStyle().Foreground(colShell).Render(string(r)))
			case '#':
				sb.WriteString(enemyStyle.Render(string(r)))
			case '|', '‾':
				sb.WriteString(playerStyle.Render(string(r)))
			case '.':
				sb.WriteString(mutedStyle.Render(string(r)))
			default:
				sb.WriteRune(r)
			}
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n")
}

// plotString writes each rune of s starting at (col,row), advancing rightward.
func plotString(g anim.RuneGrid, col, row int, s string) {
	for i, r := range []rune(s) {
		anim.PlotShell(g, col+i, row, r)
	}
}

// paneInnerWidth returns the content width for a bordered pane so it fits width.
func paneInnerWidth(width int) int {
	w := width - 4 // border (2) + padding (2)
	if w < 10 {
		w = 10
	}
	return w
}
