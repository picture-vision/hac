package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bateman/hac/internal/hass"
	"github.com/bateman/hac/internal/model"
)

type EntityDetail struct {
	entity   model.Entity
	ws       *hass.WSClient
	services []model.ServiceDomain
	width    int
	height   int
	back     bool
	scroll   int
	message  string
}

func NewEntityDetail(entity model.Entity, ws *hass.WSClient, services []model.ServiceDomain) EntityDetail {
	return EntityDetail{
		entity:   entity,
		ws:       ws,
		services: services,
	}
}

func (d *EntityDetail) SetEntity(entity model.Entity) {
	d.entity = entity
}

func (d *EntityDetail) SetSize(w, h int) {
	d.width = w
	d.height = h
}

type serviceResultMsg struct {
	err error
}

func (d EntityDetail) Update(msg tea.Msg) (EntityDetail, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			d.back = true
		case key.Matches(msg, keys.Down):
			d.scroll++
		case key.Matches(msg, keys.Up):
			if d.scroll > 0 {
				d.scroll--
			}
		case key.Matches(msg, keys.Toggle):
			return d, d.toggleEntity()
		case key.Matches(msg, keys.BrightnessUp):
			if d.entity.Domain() == "light" {
				return d, d.adjustBrightness(25)
			}
		case key.Matches(msg, keys.BrightnessDown):
			if d.entity.Domain() == "light" {
				return d, d.adjustBrightness(-25)
			}
		case key.Matches(msg, keys.RedUp):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(0, 25)
			}
		case key.Matches(msg, keys.RedDown):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(0, -25)
			}
		case key.Matches(msg, keys.GreenUp):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(1, 25)
			}
		case key.Matches(msg, keys.GreenDown):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(1, -25)
			}
		case key.Matches(msg, keys.BlueUp):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(2, 25)
			}
		case key.Matches(msg, keys.BlueDown):
			if d.entity.Domain() == "light" {
				return d, d.adjustColor(2, -25)
			}
		}
	case serviceResultMsg:
		if msg.err != nil {
			d.message = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		} else {
			d.message = successStyle.Render("OK")
		}
	}
	return d, nil
}

func (d EntityDetail) View() string {
	if d.width == 0 || d.entity.EntityID == "" {
		return ""
	}

	var b strings.Builder

	// Entity name and state
	b.WriteString(titleStyle.Render(d.entity.FriendlyName()))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(d.entity.EntityID))
	b.WriteString("\n\n")

	// State
	b.WriteString(labelStyle.Render("State: "))
	b.WriteString(stateStyle(d.entity.State).Render(d.entity.State))
	b.WriteString("\n")

	// Last changed
	b.WriteString(labelStyle.Render("Changed: "))
	b.WriteString(valueStyle.Render(d.entity.LastChanged.Format("15:04:05")))
	b.WriteString("\n")

	// Area
	if d.entity.AreaName != "" {
		b.WriteString(labelStyle.Render("Area: "))
		b.WriteString(valueStyle.Render(d.entity.AreaName))
		b.WriteString("\n")
	}

	// Quick actions
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Actions"))
	b.WriteString("\n")
	actions := d.availableActions()
	if len(actions) == 0 {
		b.WriteString(labelStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, a := range actions {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(colorSecondary).Render("["+a.key+"]"),
				valueStyle.Render(a.label)))
		}
	}

	// Message
	if d.message != "" {
		b.WriteString("\n" + d.message + "\n")
	}

	// Attributes
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Attributes"))
	b.WriteString("\n")

	attrKeys := make([]string, 0, len(d.entity.Attributes))
	for k := range d.entity.Attributes {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)

	for _, k := range attrKeys {
		v := d.entity.Attributes[k]
		val := fmt.Sprintf("%v", v)
		maxVal := d.width - len(k) - 8
		if maxVal > 0 && len(val) > maxVal {
			val = val[:maxVal-1] + "\u2026"
		}
		b.WriteString(fmt.Sprintf("  %s: %s\n",
			labelStyle.Render(k),
			valueStyle.Render(val)))
	}

	content := b.String()
	lines := strings.Split(content, "\n")
	if d.scroll > 0 && d.scroll < len(lines) {
		lines = lines[d.scroll:]
	}
	content = strings.Join(lines, "\n")

	return panelStyle.Width(d.width - 2).Height(d.height).Render(content)
}

type action struct {
	key   string
	label string
}

func (d EntityDetail) availableActions() []action {
	domain := d.entity.Domain()
	switch domain {
	case "light":
		return []action{
			{key: "t", label: "toggle"},
			{key: "+/-", label: "brightness up/down"},
			{key: "r/R", label: "red up/down"},
			{key: "f/F", label: "green up/down"},
			{key: "b/B", label: "blue up/down"},
		}
	case "switch", "input_boolean", "fan":
		return []action{
			{key: "t", label: "toggle"},
		}
	case "cover":
		return []action{
			{key: "t", label: "toggle"},
		}
	case "lock":
		return []action{
			{key: "t", label: "toggle (lock/unlock)"},
		}
	case "climate":
		return []action{
			{key: "t", label: "toggle"},
		}
	case "media_player":
		return []action{
			{key: "t", label: "toggle"},
		}
	case "scene":
		return []action{
			{key: "t", label: "activate"},
		}
	case "script":
		return []action{
			{key: "t", label: "run"},
		}
	default:
		return nil
	}
}

func (d EntityDetail) toggleEntity() tea.Cmd {
	return func() tea.Msg {
		domain := d.entity.Domain()
		service := "toggle"
		target := map[string]interface{}{
			"entity_id": d.entity.EntityID,
		}

		switch domain {
		case "scene":
			service = "turn_on"
		case "script":
			service = "turn_on"
		case "lock":
			if d.entity.State == "locked" {
				service = "unlock"
			} else {
				service = "lock"
			}
		}

		err := d.ws.CallService(domain, service, target, nil)
		return serviceResultMsg{err: err}
	}
}

func (d EntityDetail) adjustBrightness(delta int) tea.Cmd {
	return func() tea.Msg {
		current := 0.0
		if b, ok := d.entity.Attributes["brightness"]; ok {
			switch v := b.(type) {
			case float64:
				current = v
			case int:
				current = float64(v)
			}
		}

		newBrightness := int(math.Round(current)) + delta
		if newBrightness < 0 {
			newBrightness = 0
		} else if newBrightness > 255 {
			newBrightness = 255
		}

		target := map[string]interface{}{
			"entity_id": d.entity.EntityID,
		}
		data := map[string]interface{}{
			"brightness": newBrightness,
		}

		err := d.ws.CallService("light", "turn_on", target, data)
		return serviceResultMsg{err: err}
	}
}

func (d EntityDetail) adjustColor(channel, delta int) tea.Cmd {
	return func() tea.Msg {
		rgb := [3]int{255, 255, 255}
		if raw, ok := d.entity.Attributes["rgb_color"]; ok {
			if arr, ok := raw.([]interface{}); ok && len(arr) == 3 {
				for i, v := range arr {
					switch c := v.(type) {
					case float64:
						rgb[i] = int(math.Round(c))
					case int:
						rgb[i] = c
					}
				}
			}
		}

		rgb[channel] += delta
		if rgb[channel] < 0 {
			rgb[channel] = 0
		} else if rgb[channel] > 255 {
			rgb[channel] = 255
		}

		target := map[string]interface{}{
			"entity_id": d.entity.EntityID,
		}
		data := map[string]interface{}{
			"rgb_color": []int{rgb[0], rgb[1], rgb[2]},
		}

		err := d.ws.CallService("light", "turn_on", target, data)
		return serviceResultMsg{err: err}
	}
}
