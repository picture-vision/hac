package model

import "time"

type Entity struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged time.Time              `json:"last_changed"`
	LastUpdated time.Time              `json:"last_updated"`
	AreaID      string                 `json:"-"`
	AreaName    string                 `json:"-"`
}

func (e Entity) Domain() string {
	for i, c := range e.EntityID {
		if c == '.' {
			return e.EntityID[:i]
		}
	}
	return e.EntityID
}

func (e Entity) FriendlyName() string {
	if name, ok := e.Attributes["friendly_name"].(string); ok {
		return name
	}
	return e.EntityID
}

type Area struct {
	AreaID string `json:"area_id"`
	Name   string `json:"name"`
}

type EntityRegistryEntry struct {
	EntityID string `json:"entity_id"`
	AreaID   string `json:"area_id"`
	DeviceID string `json:"device_id"`
}

type DeviceRegistryEntry struct {
	ID     string `json:"id"`
	AreaID string `json:"area_id"`
}

type ServiceDomain struct {
	Domain   string             `json:"domain"`
	Services map[string]Service `json:"services"`
}

type Service struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Fields      map[string]interface{} `json:"fields"`
}

type StateChangedEvent struct {
	EntityID string `json:"entity_id"`
	OldState Entity `json:"old_state"`
	NewState Entity `json:"new_state"`
}

// ToggleServiceFor returns the appropriate service name for toggling
// an entity based on its domain and current state.
func ToggleServiceFor(domain, state string) string {
	switch domain {
	case "scene", "script":
		return "turn_on"
	case "lock":
		if state == "locked" {
			return "unlock"
		}
		return "lock"
	default:
		return "toggle"
	}
}
