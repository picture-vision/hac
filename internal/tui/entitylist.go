package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bateman/hac/internal/model"
)

type EntityList struct {
	entities  []model.Entity
	filtered  []model.Entity
	cursor    int
	offset    int
	width     int
	height    int
	searching bool
	search    textinput.Model
	groupArea bool
	selected  *model.Entity
	toggled   *model.Entity // entity to toggle (separate from selection)
}

func NewEntityList() EntityList {
	ti := textinput.New()
	ti.Placeholder = "Filter entities..."
	ti.CharLimit = 100
	return EntityList{
		search: ti,
	}
}

func (e *EntityList) SetEntities(entities []model.Entity) {
	e.entities = entities
	e.applyFilter()
}

func (e *EntityList) SetSize(w, h int) {
	e.width = w
	e.height = h
}

func (e EntityList) Update(msg tea.Msg) (EntityList, tea.Cmd) {
	if e.searching {
		return e.updateSearch(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Down):
			e.moveCursor(1)
		case key.Matches(msg, keys.Up):
			e.moveCursor(-1)
		case key.Matches(msg, keys.PageDown):
			e.moveCursor(e.visibleLines() / 2)
		case key.Matches(msg, keys.PageUp):
			e.moveCursor(-e.visibleLines() / 2)
		case key.Matches(msg, keys.Top):
			e.cursor = 0
			e.offset = 0
		case key.Matches(msg, keys.Bottom):
			e.cursor = len(e.filtered) - 1
			e.fixOffset()
		case key.Matches(msg, keys.Enter):
			if len(e.filtered) > 0 && e.cursor < len(e.filtered) {
				ent := e.filtered[e.cursor]
				e.selected = &ent
			}
		case key.Matches(msg, keys.Search):
			e.searching = true
			e.search.Focus()
			return e, e.search.Cursor.BlinkCmd()
		case key.Matches(msg, keys.Tab):
			e.groupArea = !e.groupArea
			e.applyFilter()
		case key.Matches(msg, keys.Toggle):
			if len(e.filtered) > 0 && e.cursor < len(e.filtered) {
				ent := e.filtered[e.cursor]
				e.toggled = &ent
			}
		}
	}
	return e, nil
}

func (e EntityList) updateSearch(msg tea.Msg) (EntityList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			e.searching = false
			e.search.Blur()
			e.search.SetValue("")
			e.applyFilter()
			return e, nil
		case tea.KeyEnter:
			e.searching = false
			e.search.Blur()
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.search, cmd = e.search.Update(msg)
	e.applyFilter()
	return e, cmd
}

func (e *EntityList) applyFilter() {
	query := strings.ToLower(e.search.Value())
	e.filtered = nil

	for _, ent := range e.entities {
		if query != "" {
			name := strings.ToLower(ent.FriendlyName())
			id := strings.ToLower(ent.EntityID)
			if !strings.Contains(name, query) && !strings.Contains(id, query) {
				continue
			}
		}
		e.filtered = append(e.filtered, ent)
	}

	if e.groupArea {
		sort.Slice(e.filtered, func(i, j int) bool {
			ai, aj := e.filtered[i].AreaName, e.filtered[j].AreaName
			if ai == "" {
				ai = "zzz"
			}
			if aj == "" {
				aj = "zzz"
			}
			if ai != aj {
				return ai < aj
			}
			return e.filtered[i].EntityID < e.filtered[j].EntityID
		})
	} else {
		sort.Slice(e.filtered, func(i, j int) bool {
			return e.filtered[i].EntityID < e.filtered[j].EntityID
		})
	}

	if e.cursor >= len(e.filtered) {
		e.cursor = max(0, len(e.filtered)-1)
	}
	e.fixOffset()
}

func (e EntityList) View() string {
	if e.width == 0 {
		return ""
	}

	style := activePanelStyle

	var b strings.Builder

	// Search bar
	if e.searching {
		b.WriteString(e.search.View())
		b.WriteString("\n")
	} else if e.search.Value() != "" {
		b.WriteString(labelStyle.Render(fmt.Sprintf("Filter: %s", e.search.Value())))
		b.WriteString("\n")
	}

	visibleHeight := e.visibleLines()
	if e.searching || e.search.Value() != "" {
		visibleHeight--
	}

	lastArea := ""
	lineCount := 0

	for i := e.offset; i < len(e.filtered) && lineCount < visibleHeight; i++ {
		ent := e.filtered[i]

		// Area headers
		if e.groupArea {
			area := ent.AreaName
			if area == "" {
				area = "No Area"
			}
			if area != lastArea {
				if lineCount > 0 {
					b.WriteString("\n")
					lineCount++
					if lineCount >= visibleHeight {
						break
					}
				}
				b.WriteString(areaHeaderStyle.Render(area))
				b.WriteString("\n")
				lineCount++
				lastArea = area
				if lineCount >= visibleHeight {
					break
				}
			}
		}

		name := ent.FriendlyName()
		state := stateStyle(ent.State).Render(ent.State)

		// Truncate name to fit
		maxNameWidth := e.width - lipgloss.Width(state) - 8
		if maxNameWidth < 10 {
			maxNameWidth = 10
		}
		if len(name) > maxNameWidth {
			name = name[:maxNameWidth-1] + "\u2026"
		}

		line := fmt.Sprintf("%-*s %s", maxNameWidth, name, state)

		if i == e.cursor {
			b.WriteString(selectedItemStyle.Render("> " + line))
		} else {
			b.WriteString(itemStyle.Render("  " + line))
		}
		b.WriteString("\n")
		lineCount++
	}

	// Scrollbar indicator
	total := len(e.filtered)
	info := labelStyle.Render(fmt.Sprintf(" %d/%d", min(e.cursor+1, total), total))
	if e.groupArea {
		info += labelStyle.Render(" [grouped]")
	}

	content := b.String() + "\n" + info
	return style.Width(e.width - 2).Height(e.height).Render(content)
}

func (e *EntityList) moveCursor(delta int) {
	e.cursor += delta
	if e.cursor < 0 {
		e.cursor = 0
	}
	if e.cursor >= len(e.filtered) {
		e.cursor = len(e.filtered) - 1
	}
	if e.cursor < 0 {
		e.cursor = 0
	}
	e.fixOffset()
}

func (e *EntityList) fixOffset() {
	visible := e.visibleLines()
	if e.cursor < e.offset {
		e.offset = e.cursor
	}
	if e.cursor >= e.offset+visible {
		e.offset = e.cursor - visible + 1
	}
	if e.offset < 0 {
		e.offset = 0
	}
}

func (e EntityList) visibleLines() int {
	h := e.height - 3 // padding + info line
	if h < 1 {
		h = 1
	}
	return h
}
