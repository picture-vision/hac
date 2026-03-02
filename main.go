package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bateman/hac/internal/config"
	"github.com/bateman/hac/internal/hass"
	"github.com/bateman/hac/internal/model"
	"github.com/bateman/hac/internal/tui"
)

func main() {
	// Handle init before loading config
	if len(os.Args) > 1 && os.Args[1] == "init" {
		os.Exit(cmdInit())
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		path, _ := config.Path()
		if path != "" {
			fmt.Fprintf(os.Stderr, "Run 'hac init' to create %s\n", path)
		}
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		os.Exit(runCLI(cfg, os.Args[1:]))
	}

	p := tea.NewProgram(tui.NewApp(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(cfg *config.Config, args []string) int {
	client := hass.NewRESTClient(cfg.URL, cfg.Token)

	switch args[0] {
	case "toggle":
		return cmdToggle(client, args[1:])
	case "brightness", "bri":
		return cmdBrightness(client, args[1:])
	case "red":
		return cmdColor(client, args[1:], 0)
	case "green":
		return cmdColor(client, args[1:], 1)
	case "blue":
		return cmdColor(client, args[1:], 2)
	case "call":
		return cmdCall(client, args[1:])
	case "state":
		return cmdState(client, args[1:])
	case "list", "ls":
		return cmdList(client, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printUsage()
		return 1
	}
}

// hac toggle <entity_id>
func cmdToggle(client *hass.RESTClient, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: hac toggle <entity_id>")
		return 1
	}
	entityID := args[0]

	i := strings.IndexByte(entityID, '.')
	if i < 0 {
		fmt.Fprintf(os.Stderr, "Invalid entity_id %q (expected domain.name)\n", entityID)
		return 1
	}
	domain := entityID[:i]

	service := "toggle"
	switch domain {
	case "scene":
		service = "turn_on"
	case "script":
		service = "turn_on"
	case "lock":
		entity, err := client.GetState(entityID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
			return 1
		}
		if entity.State == "locked" {
			service = "unlock"
		} else {
			service = "lock"
		}
	}

	payload := map[string]interface{}{
		"entity_id": entityID,
	}
	resp, err := client.CallService(domain, service, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Parse response — HA returns changed entity states
	var changed []model.Entity
	if err := json.Unmarshal(resp, &changed); err == nil {
		for _, e := range changed {
			if e.EntityID == entityID {
				fmt.Printf("%s: %s\n", entityID, e.State)
				return 0
			}
		}
	}

	// Verify state after toggle
	after, err := client.GetState(entityID)
	if err != nil {
		// Show raw response for debugging
		fmt.Fprintf(os.Stderr, "Response: %s\n", string(resp))
		fmt.Fprintf(os.Stderr, "State check failed: %v\n", err)
		return 1
	}
	fmt.Printf("%s: %s\n", entityID, after.State)
	return 0
}

// hac brightness <entity_id> <value|+N|-N>
func cmdBrightness(client *hass.RESTClient, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: hac brightness <entity_id> <value|+N|-N>")
		fmt.Fprintln(os.Stderr, "  value: 0-255 (absolute) or +N/-N (relative)")
		return 1
	}
	entityID := args[0]
	valStr := args[1]

	if !strings.HasPrefix(entityID, "light.") {
		fmt.Fprintf(os.Stderr, "Error: brightness only works on light entities (got %q)\n", entityID)
		return 1
	}

	var brightness int

	if strings.HasPrefix(valStr, "+") || strings.HasPrefix(valStr, "-") {
		// Relative adjustment — fetch current brightness first
		entity, err := client.GetState(entityID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
			return 1
		}

		current := 0.0
		if b, ok := entity.Attributes["brightness"]; ok {
			switch v := b.(type) {
			case float64:
				current = v
			case int:
				current = float64(v)
			}
		}

		delta, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid brightness value: %q\n", valStr)
			return 1
		}
		brightness = int(math.Round(current)) + delta
	} else {
		// Absolute value
		val, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid brightness value: %q\n", valStr)
			return 1
		}
		brightness = val
	}

	if brightness < 0 {
		brightness = 0
	} else if brightness > 255 {
		brightness = 255
	}

	payload := map[string]interface{}{
		"entity_id":  entityID,
		"brightness": brightness,
	}
	_, err := client.CallService("light", "turn_on", payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("%s: brightness %d\n", entityID, brightness)
	return 0
}

// hac red|green|blue <entity_id> <value|+N|-N>
func cmdColor(client *hass.RESTClient, args []string, channelIndex int) int {
	channelNames := []string{"red", "green", "blue"}
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: hac %s <entity_id> <value|+N|-N>\n", channelNames[channelIndex])
		fmt.Fprintln(os.Stderr, "  value: 0-255 (absolute) or +N/-N (relative)")
		return 1
	}
	entityID := args[0]
	valStr := args[1]

	if !strings.HasPrefix(entityID, "light.") {
		fmt.Fprintf(os.Stderr, "Error: %s only works on light entities (got %q)\n", channelNames[channelIndex], entityID)
		return 1
	}

	// Get current rgb_color, default [255,255,255]
	rgb := [3]int{255, 255, 255}
	entity, err := client.GetState(entityID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
		return 1
	}
	if raw, ok := entity.Attributes["rgb_color"]; ok {
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

	var value int
	if strings.HasPrefix(valStr, "+") || strings.HasPrefix(valStr, "-") {
		delta, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid %s value: %q\n", channelNames[channelIndex], valStr)
			return 1
		}
		value = rgb[channelIndex] + delta
	} else {
		val, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid %s value: %q\n", channelNames[channelIndex], valStr)
			return 1
		}
		value = val
	}

	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	rgb[channelIndex] = value

	payload := map[string]interface{}{
		"entity_id": entityID,
		"rgb_color": []int{rgb[0], rgb[1], rgb[2]},
	}
	_, err = client.CallService("light", "turn_on", payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("%s: rgb_color [%d, %d, %d]\n", entityID, rgb[0], rgb[1], rgb[2])
	return 0
}

// hac call <domain> <service> <entity_id> [json_data]
func cmdCall(client *hass.RESTClient, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: hac call <domain> <service> <entity_id> [json_data]")
		return 1
	}
	domain, service, entityID := args[0], args[1], args[2]

	payload := map[string]interface{}{
		"entity_id": entityID,
	}

	if len(args) >= 4 {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(args[3]), &extra); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid JSON data: %v\n", err)
			return 1
		}
		for k, v := range extra {
			payload[k] = v
		}
	}

	_, err := client.CallService(domain, service, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Called %s.%s on %s\n", domain, service, entityID)
	return 0
}

// hac state <entity_id>
func cmdState(client *hass.RESTClient, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: hac state <entity_id>")
		return 1
	}

	entity, err := client.GetState(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if entity.State == "off" || entity.Domain() == "light" {
		pct := 0
		if b, ok := entity.Attributes["brightness"]; ok {
			if v, ok := b.(float64); ok {
				pct = int(math.Round(v / 255 * 100))
			}
		}

		var icon string
		switch {
		case pct == 0 || entity.State == "off":
			fmt.Printf("󰹐 Off\n")
			return 0
		case pct < 30:
			icon = "󱩐"
		case pct < 60:
			icon = "󱩒"
		case pct < 100:
			icon = "󱩔"
		default:
			icon = "󰌵"
		}
		fmt.Printf("%s %d%%\n", icon, pct)
	} else {
		fmt.Println(entity.State)
	}
	return 0
}

// hac list [filter]
func cmdList(client *hass.RESTClient, args []string) int {
	entities, err := client.GetStates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(strings.Join(args, " "))
	}

	for _, e := range entities {
		if filter != "" {
			id := strings.ToLower(e.EntityID)
			name := strings.ToLower(e.FriendlyName())
			if !strings.Contains(id, filter) && !strings.Contains(name, filter) {
				continue
			}
		}
		name := e.FriendlyName()
		if name != e.EntityID {
			fmt.Printf("%-45s %-12s %s\n", e.EntityID, e.State, name)
		} else {
			fmt.Printf("%-45s %s\n", e.EntityID, e.State)
		}
	}
	return 0
}

const defaultConfig = `# Home Assistant URL (e.g. http://homeassistant.local:8123)
url: ""

# Long-lived access token
# Generate at: http://homeassistant.local:8123/profile/security
token: ""
`

// hac init
func cmdInit() int {
	path, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "Config already exists: %s\n", path)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		return 1
	}

	if err := os.WriteFile(path, []byte(defaultConfig), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		return 1
	}

	fmt.Printf("Created %s\n", path)
	fmt.Println("Edit it with your Home Assistant URL and token.")
	return 0
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `
Usage: hac [command]

Commands:
  (none)                                     Launch TUI
  init                                       Create default config file
  list [filter]                              List entities (alias: ls)
  toggle <entity_id>                         Toggle a device
  brightness <entity_id> <value|+N|-N>       Set light brightness (alias: bri)
  red <entity_id> <value|+N|-N>              Set light red channel (0-255)
  green <entity_id> <value|+N|-N>            Set light green channel (0-255)
  blue <entity_id> <value|+N|-N>             Set light blue channel (0-255)
  call <domain> <service> <entity_id> [json] Call a service
  state <entity_id>                          Print entity state

Environment variables HAC_URL and HAC_TOKEN override the config file.`)
}
