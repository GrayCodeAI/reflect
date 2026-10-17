package cmd

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// reflectBlockGlyphs — fixed 8-column ██ font used by the welcome gate banners.
var reflectBlockGlyphs = map[rune][5]string{
	'H': {"██   ██ ", "██   ██ ", "███████ ", "██   ██ ", "██   ██ "},
	'A': {"  ███   ", " █████  ", "███████ ", "██   ██ ", "██   ██ "},
	'W': {"██   ██ ", "██   ██ ", "██ █ ██ ", "███ ███ ", "██   ██ "},
	'K': {"██   ██ ", "██  ██  ", "█████   ", "██  ██  ", "██   ██ "},
	'E': {"███████ ", "██      ", "███████ ", "██      ", "███████ "},
	'L': {"██      ", "██      ", "██      ", "██      ", "███████ "},
	'C': {" ██████ ", "██    ██", "██      ", "██    ██", " ██████ "},
	'O': {" ██████ ", "██    ██", "██    ██", "██    ██", " ██████ "},
	'M': {"██    ██", "██ ██ ██", "██ ██ ██", "██    ██", "██    ██"},
	'T': {"████████", "   ██   ", "   ██   ", "   ██   ", "   ██   "},
}

const (
	reflectBlockCellW     = 8
	reflectBlockLetterGap = 1
	reflectBlockWordGap   = 4
)

// welcomeWordLines — "WELCOME" block (row-aligned, fixed grid).
var welcomeWordLines = composeReflectBlockLines("WELCOME")

// welcomeToWordLines — "TO" block, centered under WELCOME on the gate.
var welcomeToWordLines = composeReflectBlockLines("TO")

// welcomeToPhraseLines — "WELCOME TO" block for wide welcome gates.
var welcomeToPhraseLines = composeReflectBlockLines("WELCOME TO")

// welcomeToBannerMinWidth is the visible width for the WELCOME block.
const welcomeToBannerMinWidth = 61

// welcomeToPhraseMinWidth is the visible width for the combined "WELCOME TO" block.
var welcomeToPhraseMinWidth = blockLinesWidth(welcomeToPhraseLines)

func composeReflectBlockLines(text string) []string {
	rows := make([]string, 5)
	words := strings.Fields(text)
	for wi, word := range words {
		for ci, ch := range word {
			glyph, ok := reflectBlockGlyphs[ch]
			if !ok {
				continue
			}
			for i := range rows {
				if rows[i] != "" {
					if ci == 0 && wi > 0 {
						rows[i] += strings.Repeat(" ", reflectBlockWordGap)
					} else {
						rows[i] += strings.Repeat(" ", reflectBlockLetterGap)
					}
				}
				cell := padBlockCell(glyph[i], reflectBlockCellW)
				rows[i] += cell
			}
		}
	}
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return rows
}

func blockLinesWidth(lines []string) int {
	w := 0
	for _, line := range lines {
		if n := runewidth.StringWidth(line); n > w {
			w = n
		}
	}
	return w
}

func padBlockCell(s string, w int) string {
	if runewidth.StringWidth(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-runewidth.StringWidth(s))
}

// reflectRobotArtLines renders a friendly gold robot with a centered antenna and teal eyes.
// Half blocks preserve square pixels across four terminal rows.
func reflectRobotArtLines() []string {
	pixels := [...]string{
		"00011000",
		"00111100",
		"01200210",
		"01200210",
		"01111110",
		"10111101",
		"10011001",
		"00100100",
	}
	rows := make([]string, 0, len(pixels)/2)
	for y := 0; y < len(pixels); y += 2 {
		var row strings.Builder
		for x := range pixels[y] {
			switch {
			case pixels[y][x] == '2' && pixels[y+1][x] == '2':
				row.WriteString(ansiTeal + "██" + ansiOrange)
			case pixels[y][x] == '1' && pixels[y+1][x] == '1':
				row.WriteString("██")
			case pixels[y][x] == '1':
				row.WriteString("▀▀")
			case pixels[y+1][x] == '1':
				row.WriteString("▄▄")
			default:
				row.WriteString("  ")
			}
		}
		rows = append(rows, row.String())
	}
	return rows
}
