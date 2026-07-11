package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// palette translates Muster's night-ops identity into terminal-safe sRGB.
// The dark set is TAPS (the website's gunmetal default); the light set is
// REVEILLE, the same semantics on paper. Ember is the warm terminus of the
// wordmark gradient, borrowed from the site's rarity colors.
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
	ember     color.Color
	good      color.Color
	warn      color.Color
	bad       color.Color
}

func newPalette(dark, noColor bool) palette {
	if noColor {
		none := lipgloss.NoColor{}
		return palette{
			bg: none, panel: none, panelHigh: none, fg: none, muted: none,
			faint: none, line: none, lineSoft: none, accent: none, ember: none,
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
			ember:     lipgloss.Color("#E87352"),
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
		ember:     lipgloss.Color("#A63D20"),
		good:      lipgloss.Color("#137A46"),
		warn:      lipgloss.Color("#896300"),
		bad:       lipgloss.Color("#B5312C"),
	}
}

type styles struct {
	colors  palette
	noColor bool

	screen   lipgloss.Style
	title    lipgloss.Style
	subtitle lipgloss.Style
	section  lipgloss.Style
	body     lipgloss.Style
	muted    lipgloss.Style
	faint    lipgloss.Style

	selected    lipgloss.Style
	selectedBar lipgloss.Style

	// panel and focusedPanel are complete framed styles; panelBody and its
	// focused twin omit the top edge so panels can carry their title inside
	// the border line itself.
	panel             lipgloss.Style
	focusedPanel      lipgloss.Style
	panelBody         lipgloss.Style
	focusedPanelBody  lipgloss.Style
	panelTitle        lipgloss.Style
	focusedPanelTitle lipgloss.Style
	panelLine         lipgloss.Style
	focusedPanelLine  lipgloss.Style

	good    lipgloss.Style
	warn    lipgloss.Style
	bad     lipgloss.Style
	unknown lipgloss.Style

	// bottom chrome: the segmented status bar and the key ribbon under it.
	barHost    lipgloss.Style
	barMsg     lipgloss.Style
	barSection lipgloss.Style
	barAlert   lipgloss.Style
	barCounts  lipgloss.Style
	ribbonKey  lipgloss.Style
	ribbonDesc lipgloss.Style
	ribbonSep  lipgloss.Style

	dialog      lipgloss.Style
	dialogTitle lipgloss.Style
}

func newStyles(dark, noColor bool) styles {
	c := newPalette(dark, noColor)
	if noColor {
		plain := lipgloss.NewStyle()
		panel := plain.Border(lipgloss.RoundedBorder()).Padding(0, 1)
		focusedPanel := plain.Border(lipgloss.DoubleBorder()).Padding(0, 1)
		return styles{
			colors: c, noColor: true,
			screen: plain, title: plain, subtitle: plain, section: plain,
			body: plain, muted: plain, faint: plain,
			selected: plain, selectedBar: plain,
			panel: panel, focusedPanel: focusedPanel,
			panelBody:        plain.Border(lipgloss.RoundedBorder()).BorderTop(false).Padding(0, 1),
			focusedPanelBody: plain.Border(lipgloss.DoubleBorder()).BorderTop(false).Padding(0, 1),
			panelTitle:       plain, focusedPanelTitle: plain,
			panelLine: plain, focusedPanelLine: plain,
			good: plain, warn: plain, bad: plain, unknown: plain,
			barHost: plain, barMsg: plain, barSection: plain, barAlert: plain, barCounts: plain,
			ribbonKey: plain, ribbonDesc: plain, ribbonSep: plain,
			dialog:      plain.Border(lipgloss.DoubleBorder()).Padding(0, 2),
			dialogTitle: plain,
		}
	}
	return styles{
		colors: c,
		screen: lipgloss.NewStyle().
			Foreground(c.fg).
			Background(c.bg),
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
		selectedBar: lipgloss.NewStyle().
			Foreground(c.accent).
			Background(c.panelHigh),
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
		panelBody: lipgloss.NewStyle().
			Foreground(c.fg).
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(c.line).
			Padding(0, 1),
		focusedPanelBody: lipgloss.NewStyle().
			Foreground(c.fg).
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(c.accent).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Foreground(c.muted),
		focusedPanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.accent),
		panelLine: lipgloss.NewStyle().
			Foreground(c.line),
		focusedPanelLine: lipgloss.NewStyle().
			Foreground(c.accent),
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
		barHost: lipgloss.NewStyle().
			Foreground(c.muted).
			Background(c.panelHigh),
		barMsg: lipgloss.NewStyle().
			Foreground(c.muted).
			Background(c.panel),
		barSection: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.bg).
			Background(c.accent),
		barAlert: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.bg).
			Background(c.warn),
		barCounts: lipgloss.NewStyle().
			Foreground(c.muted).
			Background(c.panelHigh),
		ribbonKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.accent),
		ribbonDesc: lipgloss.NewStyle().
			Foreground(c.muted),
		ribbonSep: lipgloss.NewStyle().
			Foreground(c.faint),
		dialog: lipgloss.NewStyle().
			Foreground(c.fg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.warn).
			Padding(0, 2),
		dialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.warn),
	}
}

// gradient renders text with a per-rune accent→ember ramp — the terminal
// translation of the site's wordmark treatment. Plain text when color is off.
func (s styles) gradient(text string, bold bool) string {
	runes := []rune(text)
	if s.noColor || len(runes) == 0 {
		return text
	}
	base := lipgloss.NewStyle().Bold(bold)
	if len(runes) < 2 {
		return base.Foreground(s.colors.accent).Render(text)
	}
	stops := lipgloss.Blend1D(len(runes), s.colors.accent, s.colors.ember)
	var b strings.Builder
	for index, r := range runes {
		b.WriteString(base.Foreground(stops[index]).Render(string(r)))
	}
	return b.String()
}

// gradientRule renders a run of count copies of glyph along the same ramp.
func (s styles) gradientRule(count int, glyph string) string {
	if count <= 0 {
		return ""
	}
	if s.noColor {
		return strings.Repeat(glyph, count)
	}
	if count < 2 {
		return lipgloss.NewStyle().Foreground(s.colors.accent).Render(glyph)
	}
	stops := lipgloss.Blend1D(count, s.colors.accent, s.colors.ember)
	var b strings.Builder
	for index := 0; index < count; index++ {
		b.WriteString(lipgloss.NewStyle().Foreground(stops[index]).Render(glyph))
	}
	return b.String()
}

// spinnerStyle keeps the busy spinner on-palette wherever it is embedded.
func spinnerStyle(dark, noColor bool) lipgloss.Style {
	if noColor {
		return lipgloss.NewStyle()
	}
	c := newPalette(dark, false)
	return lipgloss.NewStyle().Foreground(c.accent).Background(c.panel)
}
