package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	Top            key.Binding
	Bottom         key.Binding
	Enter          key.Binding
	Back           key.Binding
	Search         key.Binding
	Tab            key.Binding
	Toggle         key.Binding
	Service        key.Binding
	Quit           key.Binding
	Help           key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	BrightnessUp   key.Binding
	BrightnessDown key.Binding
	RedUp          key.Binding
	RedDown        key.Binding
	GreenUp        key.Binding
	GreenDown      key.Binding
	BlueUp         key.Binding
	BlueDown       key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/\u2191", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/\u2193", "down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter", "l"),
		key.WithHelp("enter/l", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "h"),
		key.WithHelp("esc/h", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "group by area"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle"),
	),
	Service: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "services"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+u", "pgup"),
		key.WithHelp("ctrl+u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "pgdown"),
		key.WithHelp("ctrl+d", "page down"),
	),
	BrightnessUp: key.NewBinding(
		key.WithKeys("+", "="),
		key.WithHelp("+", "brightness up"),
	),
	BrightnessDown: key.NewBinding(
		key.WithKeys("-", "_"),
		key.WithHelp("-", "brightness down"),
	),
	RedUp: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "red up"),
	),
	RedDown: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "red down"),
	),
	GreenUp: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "green up"),
	),
	GreenDown: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "green down"),
	),
	BlueUp: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "blue up"),
	),
	BlueDown: key.NewBinding(
		key.WithKeys("B"),
		key.WithHelp("B", "blue down"),
	),
}
