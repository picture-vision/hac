package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bateman/hac/internal/hass"
	"github.com/bateman/hac/internal/model"
)

type serviceCallField int

const (
	fieldDomain serviceCallField = iota
	fieldService
	fieldEntity
	fieldData
	fieldCount
)

type ServiceCall struct {
	services     []model.ServiceDomain
	entities     []model.Entity
	ws           *hass.WSClient
	width        int
	height       int
	back         bool
	inputFocused bool

	// Domain selection
	domains       []string
	domainCursor  int
	domainFilter  textinput.Model
	domainPicked  string

	// Service selection
	serviceNames  []string
	serviceCursor int
	servicePicked string

	// Entity selection
	entityIDs    []string
	entityCursor int
	entityPicked string

	// Data input
	dataInput textinput.Model

	activeField serviceCallField
	message     string
	executing   bool
}

func NewServiceCall(services []model.ServiceDomain, entities []model.Entity, ws *hass.WSClient) ServiceCall {
	domains := make([]string, 0, len(services))
	for _, s := range services {
		domains = append(domains, s.Domain)
	}
	sort.Strings(domains)

	entityIDs := make([]string, 0, len(entities))
	for _, e := range entities {
		entityIDs = append(entityIDs, e.EntityID)
	}
	sort.Strings(entityIDs)

	df := textinput.New()
	df.Placeholder = "Filter domains..."
	df.CharLimit = 50

	di := textinput.New()
	di.Placeholder = `{"key": "value"}`
	di.CharLimit = 500

	return ServiceCall{
		services:     services,
		entities:     entities,
		ws:           ws,
		domains:      domains,
		entityIDs:    entityIDs,
		domainFilter: df,
		dataInput:    di,
		activeField:  fieldDomain,
	}
}

func (s *ServiceCall) SetSize(w, h int) {
	s.width = w
	s.height = h
}

func (s ServiceCall) Update(msg tea.Msg) (ServiceCall, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.inputFocused {
			return s.updateInput(msg)
		}
		switch {
		case key.Matches(msg, keys.Back):
			if s.activeField > fieldDomain {
				s.activeField--
				s.updateFieldOptions()
			} else {
				s.back = true
			}
		case key.Matches(msg, keys.Down):
			s.moveCursor(1)
		case key.Matches(msg, keys.Up):
			s.moveCursor(-1)
		case key.Matches(msg, keys.Enter):
			return s.selectCurrent()
		case key.Matches(msg, keys.Search):
			if s.activeField == fieldData {
				s.inputFocused = true
				s.dataInput.Focus()
				return s, s.dataInput.Cursor.BlinkCmd()
			}
		}
	case serviceResultMsg:
		s.executing = false
		if msg.err != nil {
			s.message = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		} else {
			s.message = successStyle.Render("Service called successfully!")
		}
	}
	return s, nil
}

func (s ServiceCall) updateInput(msg tea.KeyMsg) (ServiceCall, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		s.inputFocused = false
		s.dataInput.Blur()
		return s, nil
	case tea.KeyEnter:
		s.inputFocused = false
		s.dataInput.Blur()
		return s, nil
	}
	var cmd tea.Cmd
	s.dataInput, cmd = s.dataInput.Update(msg)
	return s, cmd
}

func (s *ServiceCall) selectCurrent() (ServiceCall, tea.Cmd) {
	switch s.activeField {
	case fieldDomain:
		filtered := s.filteredDomains()
		if s.domainCursor < len(filtered) {
			s.domainPicked = filtered[s.domainCursor]
			s.activeField = fieldService
			s.updateFieldOptions()
		}
	case fieldService:
		if s.serviceCursor < len(s.serviceNames) {
			s.servicePicked = s.serviceNames[s.serviceCursor]
			s.activeField = fieldEntity
			s.entityCursor = 0
			// Filter entities by domain
			s.updateEntityList()
		}
	case fieldEntity:
		if s.entityCursor < len(s.entityIDs) {
			s.entityPicked = s.entityIDs[s.entityCursor]
			s.activeField = fieldData
		}
	case fieldData:
		return *s, s.execute()
	}
	return *s, nil
}

func (s *ServiceCall) updateFieldOptions() {
	switch s.activeField {
	case fieldService:
		s.serviceNames = nil
		s.serviceCursor = 0
		for _, svc := range s.services {
			if svc.Domain == s.domainPicked {
				for name := range svc.Services {
					s.serviceNames = append(s.serviceNames, name)
				}
				sort.Strings(s.serviceNames)
				break
			}
		}
	}
}

func (s *ServiceCall) updateEntityList() {
	s.entityIDs = nil
	for _, e := range s.entities {
		if strings.HasPrefix(e.EntityID, s.domainPicked+".") {
			s.entityIDs = append(s.entityIDs, e.EntityID)
		}
	}
	// Also add all entities in case user wants cross-domain
	if len(s.entityIDs) == 0 {
		for _, e := range s.entities {
			s.entityIDs = append(s.entityIDs, e.EntityID)
		}
	}
	sort.Strings(s.entityIDs)
}

func (s *ServiceCall) moveCursor(delta int) {
	switch s.activeField {
	case fieldDomain:
		s.domainCursor += delta
		filtered := s.filteredDomains()
		if s.domainCursor < 0 {
			s.domainCursor = 0
		}
		if s.domainCursor >= len(filtered) {
			s.domainCursor = len(filtered) - 1
		}
	case fieldService:
		s.serviceCursor += delta
		if s.serviceCursor < 0 {
			s.serviceCursor = 0
		}
		if s.serviceCursor >= len(s.serviceNames) {
			s.serviceCursor = len(s.serviceNames) - 1
		}
	case fieldEntity:
		s.entityCursor += delta
		if s.entityCursor < 0 {
			s.entityCursor = 0
		}
		if s.entityCursor >= len(s.entityIDs) {
			s.entityCursor = len(s.entityIDs) - 1
		}
	}
}

func (s ServiceCall) filteredDomains() []string {
	return s.domains
}

func (s ServiceCall) View() string {
	if s.width == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Call Service"))
	b.WriteString("\n\n")

	// Breadcrumb
	crumbs := []string{}
	if s.domainPicked != "" {
		crumbs = append(crumbs, s.domainPicked)
	}
	if s.servicePicked != "" {
		crumbs = append(crumbs, s.servicePicked)
	}
	if s.entityPicked != "" {
		crumbs = append(crumbs, s.entityPicked)
	}
	if len(crumbs) > 0 {
		b.WriteString(labelStyle.Render(strings.Join(crumbs, " > ")))
		b.WriteString("\n\n")
	}

	visibleHeight := s.height - 8

	switch s.activeField {
	case fieldDomain:
		b.WriteString(titleStyle.Render("Select Domain"))
		b.WriteString("\n")
		domains := s.filteredDomains()
		s.renderList(&b, domains, s.domainCursor, visibleHeight)

	case fieldService:
		b.WriteString(titleStyle.Render("Select Service"))
		b.WriteString("\n")
		s.renderList(&b, s.serviceNames, s.serviceCursor, visibleHeight)

	case fieldEntity:
		b.WriteString(titleStyle.Render("Select Entity"))
		b.WriteString("\n")
		s.renderList(&b, s.entityIDs, s.entityCursor, visibleHeight)

	case fieldData:
		b.WriteString(titleStyle.Render("Service Data (JSON)"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Press / to edit, enter to execute"))
		b.WriteString("\n\n")
		b.WriteString(s.dataInput.View())
		b.WriteString("\n")
	}

	if s.message != "" {
		b.WriteString("\n" + s.message)
	}

	if s.executing {
		b.WriteString("\n" + labelStyle.Render("Executing..."))
	}

	return activePanelStyle.Width(s.width - 2).Height(s.height).Render(b.String())
}

func (s ServiceCall) renderList(b *strings.Builder, items []string, cursor int, maxVisible int) {
	offset := 0
	if cursor >= maxVisible {
		offset = cursor - maxVisible + 1
	}

	for i := offset; i < len(items) && i < offset+maxVisible; i++ {
		if i == cursor {
			b.WriteString(selectedItemStyle.Render("> " + items[i]))
		} else {
			b.WriteString(itemStyle.Render("  " + items[i]))
		}
		b.WriteString("\n")
	}

	if len(items) > maxVisible {
		b.WriteString(labelStyle.Render(fmt.Sprintf(" %d/%d", cursor+1, len(items))))
		b.WriteString("\n")
	}
}

func (s ServiceCall) execute() tea.Cmd {
	return func() tea.Msg {
		var data map[string]interface{}
		if val := s.dataInput.Value(); val != "" {
			if err := json.Unmarshal([]byte(val), &data); err != nil {
				return serviceResultMsg{err: fmt.Errorf("invalid JSON: %w", err)}
			}
		}

		target := map[string]interface{}{
			"entity_id": s.entityPicked,
		}
		err := s.ws.CallService(s.domainPicked, s.servicePicked, target, data)
		return serviceResultMsg{err: err}
	}
}
