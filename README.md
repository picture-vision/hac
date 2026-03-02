# hac

A terminal UI and CLI for [Home Assistant](https://www.home-assistant.io/).

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Interactive TUI** with vim-style navigation
- **Real-time updates** via WebSocket — entity states refresh live
- **Entity list** with search/filter and area grouping
- **Entity detail** view with quick actions (toggle, brightness, RGB color)
- **Service caller** — browse domains, services, and entities to call any HA service
- **CLI commands** for scripting — toggle, brightness, RGB color, state, list, and raw service calls
- **Light control** — brightness and per-channel RGB color adjustment (TUI keys + CLI)

## Requirements

- Go 1.25+
- A running Home Assistant instance
- A [long-lived access token](https://www.home-assistant.io/docs/authentication/#your-account-profile)

## Installation

```sh
go install github.com/picture-vision/hac@latest
```

Or build from source:

```sh
git clone https://github.com/picture-vision/hac.git
cd hac
go build -o hac .
```

## Configuration

### Quick start

```sh
hac init
```

This creates `~/.config/hac/config.yaml` with the following template:

```yaml
# Home Assistant URL (e.g. https://homeassistant.local:8123)
url: ""

# Long-lived access token
# Generate at: https://homeassistant.local:8123/profile/security
token: ""

# Allow plain HTTP connections (token sent in cleartext)
insecure: false
```

Fill in your Home Assistant URL and token, then run `hac` to launch the TUI.

### Environment variables

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `HAC_URL` | Home Assistant URL |
| `HAC_TOKEN` | Long-lived access token |
| `HAC_INSECURE` | Set to `1` or `true` to allow plain HTTP connections |

```sh
HAC_URL=https://ha.local:8123 HAC_TOKEN=ey... hac
```

### Security

By default, `hac` refuses to connect over plain `http://` because the access token would be sent in cleartext. To allow insecure connections (e.g., on a trusted local network), set `insecure: true` in the config file or `HAC_INSECURE=1` in your environment.

The config file is created with `0600` permissions and its directory with `0700`.

## Usage

### TUI

Run `hac` with no arguments to launch the interactive terminal UI.

```sh
hac
```

The TUI opens a split view with an entity list on the left. Select an entity to see its detail panel on the right. Entity states update in real time via WebSocket.

#### Keybindings

##### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move cursor down / up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Page down / up |
| `Enter` / `l` | Select entity (open detail) |
| `Esc` / `h` | Go back |

##### Entity list

| Key | Action |
|-----|--------|
| `/` | Search / filter entities |
| `Tab` | Toggle area grouping |
| `t` | Toggle selected entity on/off |

##### Entity detail (lights)

| Key | Action |
|-----|--------|
| `t` | Toggle on/off |
| `+` / `-` | Brightness up / down (+/- 25) |
| `r` / `R` | Red channel up / down (+/- 25) |
| `f` / `F` | Green channel up / down (+/- 25) |
| `b` / `B` | Blue channel up / down (+/- 25) |

##### Global

| Key | Action |
|-----|--------|
| `s` | Open service caller |
| `?` | Toggle help overlay |
| `q` / `Ctrl+c` | Quit |

### CLI

All commands use the same config file and environment variables as the TUI.

```
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
```

#### Examples

```sh
# List all light entities
hac ls light

# Toggle a switch
hac toggle switch.living_room

# Set brightness to 50%
hac brightness light.bedroom 128

# Increase brightness by 25
hac bri light.bedroom +25

# Set red channel to max
hac red light.bedroom 255

# Decrease blue channel by 50
hac blue light.bedroom -50

# Check entity state
hac state light.bedroom

# Call an arbitrary service with extra data
hac call light turn_on light.bedroom '{"color_temp": 300}'
```

Brightness and color values are 0-255. Prefix with `+` or `-` for relative adjustments (the current value is fetched automatically). Values are clamped to the 0-255 range.

## Supported entity types

The TUI provides quick actions for these entity domains:

| Domain | Actions |
|--------|---------|
| `light` | Toggle, brightness, RGB color |
| `switch`, `input_boolean`, `fan` | Toggle |
| `cover` | Toggle |
| `lock` | Toggle (lock/unlock) |
| `climate` | Toggle |
| `media_player` | Toggle |
| `scene` | Activate |
| `script` | Run |

All entity types can be controlled via the service caller (`s` key) or the `hac call` CLI command.

## Architecture

```
hac/
├── main.go              # Entry point, CLI commands
├── internal/
│   ├── config/          # YAML config + environment variable loading
│   ├── hass/
│   │   ├── rest.go      # Home Assistant REST API client
│   │   └── websocket.go # WebSocket client (real-time state, service calls)
│   ├── model/           # Shared data types (Entity, Area, Service)
│   └── tui/
│       ├── app.go       # Root Bubble Tea model, view routing
│       ├── entitylist.go    # Entity list with search/filter/grouping
│       ├── entitydetail.go  # Entity detail panel with actions
│       ├── servicecall.go   # Interactive service caller
│       ├── keys.go      # Key bindings
│       └── styles.go    # Lip Gloss styles and colors
```

The TUI uses a WebSocket connection for both real-time state updates and service calls. The CLI uses the REST API. Both transports authenticate with the same long-lived access token.

## License

MIT
