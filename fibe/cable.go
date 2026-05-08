package fibe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"nhooyr.io/websocket"
)

const (
	actionCableProtocol  = "actioncable-v1-json"
	apiKeyProtocolPrefix = "fibe-api-key."
)

type CableService struct {
	client *Client
}

type CableEvent struct {
	Resource   string          `json:"resource"`
	Message    any             `json:"message"`
	Raw        json.RawMessage `json:"raw,omitempty"`
	ReceivedAt time.Time       `json:"received_at"`
}

func (s *CableService) SubscribeResource(ctx context.Context, resource string) (<-chan CableEvent, <-chan error) {
	events := make(chan CableEvent)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		if err := s.subscribeResource(ctx, resource, events); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	return events, errs
}

func (s *CableService) subscribeResource(ctx context.Context, resource string, events chan<- CableEvent) error {
	if resource == "" {
		return fmt.Errorf("resource is required")
	}
	if s.client.cfg.apiKey == "" {
		return fmt.Errorf("api key is required")
	}

	conn, _, err := websocket.Dial(ctx, s.client.cableURL(), &websocket.DialOptions{
		Subprotocols: []string{
			actionCableProtocol,
			apiKeyProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte(s.client.cfg.apiKey)),
		},
	})
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "closing")

	identifier, err := json.Marshal(map[string]string{
		"channel":  "ApiResourceChannel",
		"resource": resource,
	})
	if err != nil {
		return err
	}
	subscribeBody, err := json.Marshal(map[string]string{
		"command":    "subscribe",
		"identifier": string(identifier),
	})
	if err != nil {
		return err
	}
	if err := conn.Write(ctx, websocket.MessageText, subscribeBody); err != nil {
		return err
	}

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		event, ok, err := parseCableFrame(resource, data)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- event:
		}
	}
}

func (c *Client) cableURL() string {
	u, err := url.Parse(c.cfg.baseURL())
	if err != nil {
		return c.cfg.baseURL() + "/cable"
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "wss"
	}
	u.Path = "/cable"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func parseCableFrame(resource string, data []byte) (CableEvent, bool, error) {
	var frame struct {
		Type    string          `json:"type"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		return CableEvent{}, false, err
	}
	switch frame.Type {
	case "welcome", "ping", "confirm_subscription":
		return CableEvent{}, false, nil
	case "reject_subscription":
		return CableEvent{}, false, fmt.Errorf("cable subscription rejected")
	}
	if len(frame.Message) == 0 || string(frame.Message) == "null" {
		return CableEvent{}, false, nil
	}
	var message any
	if err := json.Unmarshal(frame.Message, &message); err != nil {
		return CableEvent{}, false, err
	}
	return CableEvent{
		Resource:   resource,
		Message:    message,
		Raw:        append(json.RawMessage(nil), frame.Message...),
		ReceivedAt: time.Now().UTC(),
	}, true, nil
}
