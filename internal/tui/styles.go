package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// palette translates the explainer video's night-ops theme into terminal-safe
// sRGB colors. The light set preserves the same semantic contrast for terminals
// that report a light background.
type palette struct {
	bg        color.Color
	panel     color.Color
	panelHigh color.Color
	fg        color.Color
	muted     color.Color
	faint     color.Color
	line      color.Color
	lineSoft  color.Color
	accent    color.Color
	good      color.Color
	warn      color.Color
	bad       color.Color
}

func newPalette(dark, noColor bool) palette {
	if noColor {
		none := lipgloss.NoColor{}
		return palette{
			bg: none, panel: none, panelHigh: none, fg: none, muted: none,
			faint: none, line: none, lineSoft: none, accent: none,
			good: none, warn: none, bad: none,
		}
	}
	if dark {
		return palette{
			bg:        lipgloss.Color("#22252B"),
			panel:     lipgloss.Color("#292D34"),
			panelHigh: lipgloss.Color("#30353D"),
			fg:        lipgloss.Color("#F1F0EB"),
			muted:     lipgloss.Color("#A1A5AE"),
			faint:     lipgloss.Color("#6D727C"),
			line:      lipgloss.Color("#454A54"),
			lineSoft:  lipgloss.Color("#373B43"),
			accent:    lipgloss.Color("#E7A64A"),
			good:      lipgloss.Color("#52C57C"),
			warn:      lipgloss.Color("#DDBC52"),
			bad:       lipgloss.Color("#E26450"),
		}
	}
	return palette{
		bg:        lipgloss.Color("#F8F5EE"),
		panel:     lipgloss.Color("#EEE9DF"),
		panelHigh: lipgloss.Color("#E5DED1"),
		fg:        lipgloss.Color("#25282F"),
		muted:     lipgloss.Color("#666C76"),
		faint:     lipgloss.Color("#858A91"),
		line:      lipgloss.Color("#C8C0B3"),
		lineSoft:  lipgloss.Color("#DDD6CA"),
		accent:    lipgloss.Color("#A95C00"),
		good:      lipgloss.Color("#137A46"),
		warn:      lipgloss.Color("#896300"),
		bad:       lipgloss.Color("#B5312C"),
	}
}

type styles struct {
	colors       palette
	screen       lipgloss.Style
	brand        lipgloss.Style
	eyebrow      lipgloss.Style
	title        lipgloss.Style
	subtitle     lipgloss.Style
	section      lipgloss.Style
	body         lipgloss.Style
	muted        lipgloss.Style
	faint        lipgloss.Style
	selected     lipgloss.Style
	key          lipgloss.Style
	panel        lipgloss.Style
	focusedPanel lipgloss.Style
	good         lipgloss.Style
	warn         lipgloss.Style
	bad          lipgloss.Style
	unknown      lipgloss.Style
}

func newStyles(dark, noColor bool) styles {
	c := newPalette(dark, noColor)
	if noColor {
		plain := lipgloss.NewStyle()
		panel := plain.Border(lipgloss.RoundedBorder()).Padding(0, 1)
		focusedPanel := plain.Border(lipgloss.DoubleBorder()).Padding(0, 1)
		return styles{
			colors: c, screen: plain, brand: plain, eyebrow: plain, title: plain,
			subtitle: plain, section: plain, body: plain, muted: plain, faint: plain,
			selected: plain, key: plain, panel: panel, focusedPanel: focusedPanel,
			good: plain, warn: plain, bad: plain, unknown: plain,
		}
	}
	return styles{
		colors: c,
		screen: lipgloss.NewStyle().
			Foreground(c.fg).
			Background(c.bg),
		brand: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.bg).
			Background(c.accent).
			Padding(0, 1),
		eyebrow: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.accent),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.fg),
		subtitle: lipgloss.NewStyle().
			Foreground(c.muted),
		section: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.accent),
		body: lipgloss.NewStyle().
			Foreground(c.fg),
		muted: lipgloss.NewStyle().
			Foreground(c.muted),
		faint: lipgloss.NewStyle().
			Foreground(c.faint),
		selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.fg).
			Background(c.panelHigh),
		key: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.accent),
		panel: lipgloss.NewStyle().
			Foreground(c.fg).
			Background(c.panel).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.line).
			Padding(0, 1),
		focusedPanel: lipgloss.NewStyle().
			Foreground(c.fg).
			Background(c.panel).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.accent).
			Padding(0, 1),
		good: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.good),
		warn: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.warn),
		bad: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.bad),
		unknown: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.muted),
	}
}
