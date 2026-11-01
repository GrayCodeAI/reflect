package cmd

import (
	"os"
	"strings"
)

// The mark is drawn on three lines, then reflected across a quiet waterline.
// Geometry stays fixed so the welcome never shimmers or shifts while typing.
func renderWaterWelcomeArt(width, height int) string {
	if width < 8 || height < 12 {
		return ""
	}
	plainTerminal := os.Getenv("TERM") == "dumb"
	_, noColor := os.LookupEnv("NO_COLOR")
	paint := func(color, text string) string {
		if noColor || plainTerminal {
			return text
		}
		return color + text + ansiReset
	}
	if width < 27 || height < 20 {
		name := "R E F L E C T"
		if width < len(name) {
			name = "REFLECT"
		}
		return paint(ansiBold+ansiOrange, name) + "\n" + paint(ansiTeal, strings.Repeat("~", len(name)))
	}
	glyphs := [][3]string{
		{"╭─╮", "├┬╯", "╵╰╴"}, // R
		{"╭──", "├─ ", "╰──"}, // E
		{"╭──", "├─ ", "╵  "}, // F
		{"╷  ", "│  ", "╰──"}, // L
		{"╭──", "├─ ", "╰──"}, // E
		{"╭──", "│  ", "╰──"}, // C
		{"─┬─", " │ ", " ╵ "}, // T
	}
	if plainTerminal {
		glyphs = [][3]string{
			{".-.", "|-'", `| \`},
			{".--", "|--", "'--"},
			{".--", "|--", "|  "},
			{"|  ", "|  ", "'--"},
			{".--", "|--", "'--"},
			{".--", "|  ", "'--"},
			{"-+-", " | ", " ' "},
		}
	}
	gap := "  "
	if width < 33 {
		gap = " "
	}
	var mark [3]string
	for row := range mark {
		parts := make([]string, len(glyphs))
		for i, glyph := range glyphs {
			parts[i] = glyph[row]
		}
		mark[row] = strings.Join(parts, gap)
	}
	rows := make([]string, 0, 6)
	for _, row := range mark {
		rows = append(rows, paint(ansiBold+ansiOrange, row))
	}
	markWidth := len([]rune(mark[0]))
	left := (markWidth - 5) / 2
	rule := strings.Repeat("─", left) + "  ~  " + strings.Repeat("─", markWidth-left-5)
	if plainTerminal {
		rule = strings.ReplaceAll(rule, "─", "-")
	}
	rows = append(rows, paint("\x1b[38;2;79;138;151m", rule))
	flip := strings.NewReplacer("╭", "╰", "╰", "╭", "╮", "╯", "╯", "╮", "╷", "╵", "╵", "╷", "┬", "┴", "┴", "┬", ".", "'", "'", ".", "\\", "/", "/", "\\")
	rows = append(rows, paint("\x1b[38;2;52;87;98m", flip.Replace(mark[2])))
	if height >= 24 {
		rows = append(rows, paint("\x1b[38;2;35;57;66m", flip.Replace(mark[1])))
	}
	return strings.Join(rows, "\n")
}
