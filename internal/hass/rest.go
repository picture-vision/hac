package hass

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bateman/hac/internal/model"
)

type RESTClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewRESTClient(baseURL, token string) *RESTClient {
	return &RESTClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{},
	}
}

func (c *RESTClient) do(method, path string, body interface{}) ([]byte, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		// Try to extract HA's error message from JSON response
		var haErr struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &haErr) == nil && haErr.Message != "" {
			return nil, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, haErr.Message)
		}
		return nil, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *RESTClient) GetStates() ([]model.Entity, error) {
	data, err := c.do("GET", "/api/states", nil)
	if err != nil {
		return nil, err
	}
	var entities []model.Entity
	return entities, json.Unmarshal(data, &entities)
}

func (c *RESTClient) GetServices() ([]model.ServiceDomain, error) {
	data, err := c.do("GET", "/api/services", nil)
	if err != nil {
		return nil, err
	}
	var services []model.ServiceDomain
	return services, json.Unmarshal(data, &services)
}

func (c *RESTClient) GetState(entityID string) (*model.Entity, error) {
	data, err := c.do("GET", fmt.Sprintf("/api/states/%s", entityID), nil)
	if err != nil {
		return nil, err
	}
	var entity model.Entity
	return &entity, json.Unmarshal(data, &entity)
}

func (c *RESTClient) CallService(domain, service string, payload map[string]interface{}) ([]byte, error) {
	return c.do("POST", fmt.Sprintf("/api/services/%s/%s", domain, service), payload)
}
