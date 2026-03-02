package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bateman/hac/internal/config"
	"github.com/bateman/hac/internal/hass"
	"github.com/bateman/hac/internal/model"
)

type view int

const (
	viewEntityList view = iota
	viewEntityDetail
	viewServiceCall
)

type App struct {
	cfg    *config.Config
	ws     *hass.WSClient
	width  int
	height int

	activeView   view
	entityList   EntityList
	entityDetail EntityDetail
	serviceCall  ServiceCall

	entities []model.Entity
	areas    []model.Area
	services []model.ServiceDomain

	connected bool
	err       error
	showHelp  bool
}

// Bubble Tea messages
type wsMsg struct {
	msg interface{}
}

func NewApp(cfg *config.Config) App {
	ws := hass.NewWSClient(cfg.URL, cfg.Token)
	return App{
		cfg:        cfg,
		ws:         ws,
		entityList: NewEntityList(),
	}
}

func (a App) Init() tea.Cmd {
	a.ws.Connect()
	return tea.Batch(
		listenWS(a.ws),
	)
}

func listenWS(ws *hass.WSClient) tea.Cmd {
	return func() tea.Msg {
		msg := <-ws.Messages()
		return wsMsg{msg: msg}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resizeViews()

	case tea.KeyMsg:
		// Global keys
		if a.entityList.searching {
			// Pass to entity list when searching
			break
		}
		if a.activeView == viewServiceCall && a.serviceCall.inputFocused {
			break
		}
		switch {
		case key.Matches(msg, keys.Quit):
			a.ws.Close()
			return a, tea.Quit
		case key.Matches(msg, keys.Help):
			a.showHelp = !a.showHelp
			return a, nil
		case key.Matches(msg, keys.Service):
			if a.activeView != viewServiceCall {
				a.activeView = viewServiceCall
				a.serviceCall = NewServiceCall(a.services, a.entities, a.ws)
				a.serviceCall.SetSize(a.width-2, a.contentHeight())
				return a, nil
			}
		}

	case wsMsg:
		cmds = append(cmds, listenWS(a.ws))
		switch m := msg.msg.(type) {
		case hass.WSConnected:
			a.connected = true
			a.err = nil
			a.entities = m.Entities
			a.areas = m.Areas
			a.services = m.Services
			a.entityList.SetEntities(a.entities)
			a.resizeViews()
		case hass.WSStateChanged:
			for i, e := range a.entities {
				if e.EntityID == m.Entity.EntityID {
					// Preserve area info
					m.Entity.AreaID = e.AreaID
					m.Entity.AreaName = e.AreaName
					a.entities[i] = m.Entity
					break
				}
			}
			a.entityList.SetEntities(a.entities)
			if a.activeView == viewEntityDetail && a.entityDetail.entity.EntityID == m.Entity.EntityID {
				a.entityDetail.SetEntity(m.Entity)
			}
		case hass.WSError:
			a.err = m.Err
		case hass.WSDisconnected:
			a.connected = false
		}
		return a, tea.Batch(cmds...)
	}

	// Route to active view
	switch a.activeView {
	case viewEntityList:
		var cmd tea.Cmd
		a.entityList, cmd = a.entityList.Update(msg)
		cmds = append(cmds, cmd)
		if a.entityList.toggled != nil {
			cmds = append(cmds, a.toggleEntity(*a.entityList.toggled))
			a.entityList.toggled = nil
		}
		if a.entityList.selected != nil {
			a.activeView = viewEntityDetail
			a.entityDetail = NewEntityDetail(*a.entityList.selected, a.ws, a.services)
			a.entityList.selected = nil
			a.resizeViews()
		}
	case viewEntityDetail:
		var cmd tea.Cmd
		a.entityDetail, cmd = a.entityDetail.Update(msg)
		cmds = append(cmds, cmd)
		if a.entityDetail.back {
			a.activeView = viewEntityList
			a.resizeViews()
		}
	case viewServiceCall:
		var cmd tea.Cmd
		a.serviceCall, cmd = a.serviceCall.Update(msg)
		cmds = append(cmds, cmd)
		if a.serviceCall.back {
			a.activeView = viewEntityList
			a.resizeViews()
		}
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	header := a.renderHeader()
	var content string

	switch a.activeView {
	case viewEntityList:
		list := a.entityList.View()
		if a.entityDetail.entity.EntityID != "" {
			detail := a.entityDetail.View()
			content = lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
		} else {
			content = list
		}
	case viewEntityDetail:
		list := a.entityList.View()
		detail := a.entityDetail.View()
		content = lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
	case viewServiceCall:
		content = a.serviceCall.View()
	}

	statusBar := a.renderStatusBar()

	if a.showHelp {
		content = a.renderHelp()
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

func (a App) renderHeader() string {
	status := "disconnected"
	statusSt := errorStyle
	if a.connected {
		status = "connected"
		statusSt = successStyle
	}

	title := headerStyle.Render("hac")
	st := statusSt.Render(status)
	entityCount := labelStyle.Render(fmt.Sprintf("%d entities", len(a.entities)))

	gap := a.width - lipgloss.Width(title) - lipgloss.Width(st) - lipgloss.Width(entityCount) - 2
	if gap < 0 {
		gap = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		title,
		lipgloss.NewStyle().Width(gap).Render(""),
		entityCount,
		" ",
		st,
	)
}

func (a App) renderStatusBar() string {
	if a.err != nil {
		return statusBarStyle.Foreground(colorError).Render(fmt.Sprintf("Error: %v", a.err))
	}

	help := "? help  / search  tab group  t toggle  s services  q quit"
	return helpStyle.Render(help)
}

func (a App) renderHelp() string {
	help := titleStyle.Render("Keybindings") + "\n\n"
	help += fmt.Sprintf("  %s  %s\n", labelStyle.Render("j/k"), valueStyle.Render("navigate up/down"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("g/G"), valueStyle.Render("go to top/bottom"))
	help += fmt.Sprintf("  %s  %s\n", labelStyle.Render("enter"), valueStyle.Render("select entity"))
	help += fmt.Sprintf("  %s  %s\n", labelStyle.Render("esc/h"), valueStyle.Render("go back"))
	help += fmt.Sprintf("  %s      %s\n", labelStyle.Render("/"), valueStyle.Render("search/filter"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("tab"), valueStyle.Render("toggle area grouping"))
	help += fmt.Sprintf("  %s      %s\n", labelStyle.Render("t"), valueStyle.Render("toggle entity on/off"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("+/-"), valueStyle.Render("brightness up/down (lights)"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("r/R"), valueStyle.Render("red up/down (lights)"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("f/F"), valueStyle.Render("green up/down (lights)"))
	help += fmt.Sprintf("  %s    %s\n", labelStyle.Render("b/B"), valueStyle.Render("blue up/down (lights)"))
	help += fmt.Sprintf("  %s      %s\n", labelStyle.Render("s"), valueStyle.Render("open service caller"))
	help += fmt.Sprintf("  %s      %s\n", labelStyle.Render("?"), valueStyle.Render("toggle this help"))
	help += fmt.Sprintf("  %s      %s\n", labelStyle.Render("q"), valueStyle.Render("quit"))
	return panelStyle.Width(a.width - 2).Height(a.contentHeight()).Render(help)
}

func (a *App) resizeViews() {
	a.entityList.SetSize(a.listWidth(), a.contentHeight())
	a.entityDetail.SetSize(a.detailWidth(), a.contentHeight())
	a.serviceCall.SetSize(a.width-2, a.contentHeight())
}

func (a App) toggleEntity(entity model.Entity) tea.Cmd {
	return func() tea.Msg {
		domain := entity.Domain()
		service := model.ToggleServiceFor(domain, entity.State)
		target := map[string]interface{}{
			"entity_id": entity.EntityID,
		}

		err := a.ws.CallService(domain, service, target, nil)
		return serviceResultMsg{err: err}
	}
}

func (a App) contentHeight() int {
	return a.height - 3 // header + status bar + padding
}

func (a App) listWidth() int {
	if a.activeView == viewEntityDetail || a.entityDetail.entity.EntityID != "" {
		return a.width / 2
	}
	return a.width
}

func (a App) detailWidth() int {
	return a.width - a.listWidth()
}
