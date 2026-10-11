package cmd

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// decoderBlockGlyphs — fixed 8-column ██ font used by the welcome gate banners.
var decoderBlockGlyphs = map[rune][5]string{
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

// decoderLogoArtLines is the canonical portable ASCII DECODER wordmark.
var decoderLogoArtLines = []string{
	" ____  _____ ____ ___  ____  _____ ____  ",
	"|  _ \\| ____/ ___/ _ \\|  _ \\| ____|  _ \\ ",
	"| | | |  _|| |  | | | | | | |  _| | |_) |",
	"| |_| | |__| |__| |_| | |_| | |___|  _ < ",
	"|____/|_____\\____\\___/|____/|_____|_| \\_\\",
}

const (
	decoderBlockCellW     = 8
	decoderBlockLetterGap = 1
	decoderBlockWordGap   = 4
)

// welcomeWordLines — "WELCOME" block (row-aligned, fixed grid).
var welcomeWordLines = composeDecoderBlockLines("WELCOME")

// welcomeToWordLines — "TO" block, centered under WELCOME on the gate.
var welcomeToWordLines = composeDecoderBlockLines("TO")

// welcomeToPhraseLines — "WELCOME TO" block for wide welcome gates.
var welcomeToPhraseLines = composeDecoderBlockLines("WELCOME TO")

// welcomeToBannerMinWidth is the visible width for the WELCOME block.
const welcomeToBannerMinWidth = 61

// welcomeToPhraseMinWidth is the visible width for the combined "WELCOME TO" block.
var welcomeToPhraseMinWidth = blockLinesWidth(welcomeToPhraseLines)

func composeDecoderBlockLines(text string) []string {
	rows := make([]string, 5)
	words := strings.Fields(text)
	for wi, word := range words {
		for ci, ch := range word {
			glyph, ok := decoderBlockGlyphs[ch]
			if !ok {
				continue
			}
			for i := range rows {
				if rows[i] != "" {
					if ci == 0 && wi > 0 {
						rows[i] += strings.Repeat(" ", decoderBlockWordGap)
					} else {
						rows[i] += strings.Repeat(" ", decoderBlockLetterGap)
					}
				}
				cell := padBlockCell(glyph[i], decoderBlockCellW)
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
