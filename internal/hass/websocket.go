package hass

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/bateman/hac/internal/model"
)

type WSMessage struct {
	ID      int             `json:"id,omitempty"`
	Type    string          `json:"type"`
	Success *bool           `json:"success,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Event   *WSEvent        `json:"event,omitempty"`

	// Auth fields
	AccessToken string `json:"access_token,omitempty"`

	// Subscribe fields
	EventType string `json:"event_type,omitempty"`

	// Command fields
	Domain  string                 `json:"domain,omitempty"`
	Service string                 `json:"service,omitempty"`
	Target  map[string]interface{} `json:"target,omitempty"`
	Data    map[string]interface{} `json:"service_data,omitempty"`
}

type WSEvent struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// Messages sent from WebSocket client to Bubble Tea
type WSStateChanged struct {
	Entity model.Entity
}

type WSConnected struct {
	Entities []model.Entity
	Areas    []model.Area
	Services []model.ServiceDomain
}

type WSError struct {
	Err error
}

type WSDisconnected struct{}

type WSClient struct {
	url   string
	token string
	msgCh chan interface{} // sends messages to Bubble Tea

	conn    *websocket.Conn
	mu      sync.Mutex
	writeMu sync.Mutex // protects websocket writes
	nextID  atomic.Int64
	done    chan struct{}

	// pending response handlers
	pending   map[int]chan WSMessage
	pendingMu sync.Mutex
}

func NewWSClient(baseURL, token string) *WSClient {
	u, _ := url.Parse(baseURL)
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/api/websocket", scheme, u.Host)

	return &WSClient{
		url:     wsURL,
		token:   token,
		msgCh:   make(chan interface{}, 64),
		pending: make(map[int]chan WSMessage),
	}
}

func (c *WSClient) Messages() <-chan interface{} {
	return c.msgCh
}

func (c *WSClient) Connect() {
	go c.connectLoop()
}

func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done != nil {
		close(c.done)
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *WSClient) connectLoop() {
	backoff := time.Second
	for {
		err := c.runConnection()
		if err != nil {
			c.send(WSError{Err: err})
		}
		c.send(WSDisconnected{})

		// Check if explicitly closed
		c.mu.Lock()
		if c.done != nil {
			select {
			case <-c.done:
				c.mu.Unlock()
				return
			default:
			}
		}
		c.mu.Unlock()

		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *WSClient) runConnection() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.done = make(chan struct{})
	c.nextID.Store(1)
	c.mu.Unlock()

	defer func() {
		conn.Close()
		c.pendingMu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = make(map[int]chan WSMessage)
		c.pendingMu.Unlock()
	}()

	// Read auth_required
	var msg WSMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("read auth_required: %w", err)
	}

	// Send auth
	auth := WSMessage{
		Type:        "auth",
		AccessToken: c.token,
	}
	if err := conn.WriteJSON(auth); err != nil {
		return fmt.Errorf("write auth: %w", err)
	}

	// Read auth result
	if err := conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("read auth_ok: %w", err)
	}
	if msg.Type != "auth_ok" {
		return fmt.Errorf("auth failed: %s", msg.Type)
	}

	// Fetch initial data
	if err := c.fetchInitialData(conn); err != nil {
		return fmt.Errorf("initial data: %w", err)
	}

	// Subscribe to state changes
	subID := int(c.nextID.Add(1) - 1)
	sub := WSMessage{
		ID:        subID,
		Type:      "subscribe_events",
		EventType: "state_changed",
	}
	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// Read loop
	for {
		var raw WSMessage
		if err := conn.ReadJSON(&raw); err != nil {
			return fmt.Errorf("read: %w", err)
		}

		// Route responses to pending handlers
		if raw.ID != 0 && raw.Type == "result" {
			c.pendingMu.Lock()
			ch, ok := c.pending[raw.ID]
			if ok {
				delete(c.pending, raw.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- raw
			}
			continue
		}

		// Handle events
		if raw.Type == "event" && raw.Event != nil && raw.Event.EventType == "state_changed" {
			var sc model.StateChangedEvent
			if err := json.Unmarshal(raw.Event.Data, &sc); err == nil {
				c.send(WSStateChanged{Entity: sc.NewState})
			}
		}
	}
}

func (c *WSClient) fetchInitialData(conn *websocket.Conn) error {
	type result struct {
		entities []model.Entity
		areas    []model.Area
		services []model.ServiceDomain
		err      error
	}

	// Get states
	statesID := int(c.nextID.Add(1) - 1)
	statesCh := c.registerPending(statesID)
	if err := conn.WriteJSON(WSMessage{ID: statesID, Type: "get_states"}); err != nil {
		return err
	}

	// Get areas
	areasID := int(c.nextID.Add(1) - 1)
	areasCh := c.registerPending(areasID)
	if err := conn.WriteJSON(WSMessage{
		ID:   areasID,
		Type: "config/area_registry/list",
	}); err != nil {
		return err
	}

	// Get entity registry
	entityRegID := int(c.nextID.Add(1) - 1)
	entityRegCh := c.registerPending(entityRegID)
	if err := conn.WriteJSON(WSMessage{
		ID:   entityRegID,
		Type: "config/entity_registry/list",
	}); err != nil {
		return err
	}

	// Get device registry
	deviceRegID := int(c.nextID.Add(1) - 1)
	deviceRegCh := c.registerPending(deviceRegID)
	if err := conn.WriteJSON(WSMessage{
		ID:   deviceRegID,
		Type: "config/device_registry/list",
	}); err != nil {
		return err
	}

	// Collect responses (read from conn until we have all 4)
	received := 0
	for received < 4 {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		if msg.Type == "result" {
			c.pendingMu.Lock()
			ch, ok := c.pending[msg.ID]
			if ok {
				delete(c.pending, msg.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- msg
				received++
			}
		}
	}

	// Parse states
	statesMsg := <-statesCh
	var entities []model.Entity
	if err := json.Unmarshal(statesMsg.Result, &entities); err != nil {
		return fmt.Errorf("parse states: %w", err)
	}

	// Parse areas
	areasMsg := <-areasCh
	var areas []model.Area
	if err := json.Unmarshal(areasMsg.Result, &areas); err != nil {
		return fmt.Errorf("parse areas: %w", err)
	}
	areaMap := make(map[string]string)
	for _, a := range areas {
		areaMap[a.AreaID] = a.Name
	}

	// Parse entity registry
	entityRegMsg := <-entityRegCh
	var entityReg []model.EntityRegistryEntry
	if err := json.Unmarshal(entityRegMsg.Result, &entityReg); err != nil {
		return fmt.Errorf("parse entity registry: %w", err)
	}
	entityAreaMap := make(map[string]string) // entity_id -> area_id
	entityDeviceMap := make(map[string]string) // entity_id -> device_id
	for _, e := range entityReg {
		if e.AreaID != "" {
			entityAreaMap[e.EntityID] = e.AreaID
		}
		if e.DeviceID != "" {
			entityDeviceMap[e.EntityID] = e.DeviceID
		}
	}

	// Parse device registry (for entities that inherit area from device)
	deviceRegMsg := <-deviceRegCh
	var deviceReg []model.DeviceRegistryEntry
	if err := json.Unmarshal(deviceRegMsg.Result, &deviceReg); err != nil {
		return fmt.Errorf("parse device registry: %w", err)
	}
	deviceAreaMap := make(map[string]string) // device_id -> area_id
	for _, d := range deviceReg {
		if d.AreaID != "" {
			deviceAreaMap[d.ID] = d.AreaID
		}
	}

	// Assign areas to entities
	for i := range entities {
		eid := entities[i].EntityID
		areaID := entityAreaMap[eid]
		if areaID == "" {
			if devID := entityDeviceMap[eid]; devID != "" {
				areaID = deviceAreaMap[devID]
			}
		}
		if areaID != "" {
			entities[i].AreaID = areaID
			entities[i].AreaName = areaMap[areaID]
		}
	}

	// Get services via REST-style WS isn't straightforward, we'll parse from states
	// Actually HA has get_services
	svcID := int(c.nextID.Add(1) - 1)
	svcCh := c.registerPending(svcID)
	if err := conn.WriteJSON(WSMessage{ID: svcID, Type: "get_services"}); err != nil {
		return err
	}

	// Read until we get the service result
	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		if msg.Type == "result" && msg.ID == svcID {
			c.pendingMu.Lock()
			ch, ok := c.pending[msg.ID]
			if ok {
				delete(c.pending, msg.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- msg
			}
			break
		}
	}

	svcMsg := <-svcCh
	var services []model.ServiceDomain
	// HA returns services as a map of domain -> {services: ...}
	var svcMap map[string]struct {
		Services map[string]model.Service `json:"services"`
	}
	if err := json.Unmarshal(svcMsg.Result, &svcMap); err != nil {
		// Try as flat map
		var flatMap map[string]map[string]model.Service
		if err2 := json.Unmarshal(svcMsg.Result, &flatMap); err2 == nil {
			for domain, svcs := range flatMap {
				services = append(services, model.ServiceDomain{Domain: domain, Services: svcs})
			}
		}
	} else {
		for domain, s := range svcMap {
			services = append(services, model.ServiceDomain{Domain: domain, Services: s.Services})
		}
	}

	c.send(WSConnected{
		Entities: entities,
		Areas:    areas,
		Services: services,
	})

	return nil
}

func (c *WSClient) registerPending(id int) chan WSMessage {
	ch := make(chan WSMessage, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()
	return ch
}

func (c *WSClient) send(msg interface{}) {
	select {
	case c.msgCh <- msg:
	default:
		// Drop if full
	}
}

func (c *WSClient) CallService(domain, service string, target map[string]interface{}, data map[string]interface{}) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	id := int(c.nextID.Add(1) - 1)
	ch := c.registerPending(id)

	msg := WSMessage{
		ID:      id,
		Type:    "call_service",
		Domain:  domain,
		Service: service,
		Target:  target,
		Data:    data,
	}
	c.writeMu.Lock()
	err := conn.WriteJSON(msg)
	c.writeMu.Unlock()
	if err != nil {
		return err
	}

	select {
	case resp := <-ch:
		if resp.Success != nil && !*resp.Success {
			return fmt.Errorf("service call failed")
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout")
	}
}

// WSURLFromHTTP converts an HTTP URL to a WebSocket URL
func WSURLFromHTTP(httpURL string) string {
	if strings.HasPrefix(httpURL, "https") {
		return strings.Replace(httpURL, "https", "wss", 1) + "/api/websocket"
	}
	return strings.Replace(httpURL, "http", "ws", 1) + "/api/websocket"
}
