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
		if resource == "" {
			errs <- fmt.Errorf("resource is required")
			return
		}
		identifier := map[string]any{
			"channel":  "ApiResourceChannel",
			"resource": resource,
		}
		if err := s.subscribeCable(ctx, resource, identifier, 0, events); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	return events, errs
}

func (s *CableService) SubscribeLogStream(ctx context.Context, playgroundID int64, service string, opts *LogsStreamOptions) (<-chan LogStreamEvent, <-chan error) {
	events := make(chan LogStreamEvent, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		if err := s.subscribeLogStream(ctx, playgroundID, service, opts, events); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	return events, errs
}

func (s *CableService) subscribeLogStream(ctx context.Context, playgroundID int64, service string, opts *LogsStreamOptions, events chan<- LogStreamEvent) error {
	if playgroundID <= 0 {
		return fmt.Errorf("playground id is required")
	}
	identifier := map[string]any{
		"channel":       "PlaygroundLogsChannel",
		"playground_id": playgroundID,
		"subscriber_id": logSubscriberID(),
	}
	if service != "" {
		identifier["channel"] = "ContainerLogsChannel"
		identifier["service_name"] = service
	}
	if opts != nil && opts.Tail > 0 {
		identifier["tail"] = opts.Tail
	}

	cableEvents := make(chan CableEvent, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(cableEvents)
		defer close(errs)
		if err := s.subscribeCable(ctx, "LogStream", identifier, 15*time.Second, cableEvents); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()

	for cableEvents != nil || errs != nil {
		select {
		case ev, ok := <-cableEvents:
			if !ok {
				cableEvents = nil
				continue
			}
			var logEvent LogStreamEvent
			if err := json.Unmarshal(ev.Raw, &logEvent); err != nil {
				return err
			}
			logEvent.ReceivedAt = ev.ReceivedAt
			select {
			case events <- logEvent:
			case <-ctx.Done():
				return ctx.Err()
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (s *CableService) subscribeCable(ctx context.Context, label string, identifier map[string]any, heartbeatInterval time.Duration, events chan<- CableEvent) error {
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

	identifierJSON, err := json.Marshal(identifier)
	if err != nil {
		return err
	}
	subscribeBody, err := json.Marshal(map[string]string{
		"command":    "subscribe",
		"identifier": string(identifierJSON),
	})
	if err != nil {
		return err
	}
	if err := conn.Write(ctx, websocket.MessageText, subscribeBody); err != nil {
		return err
	}
	if heartbeatInterval > 0 {
		done := make(chan struct{})
		defer close(done)
		go sendCableHeartbeats(ctx, conn, string(identifierJSON), heartbeatInterval, done)
	}

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return err
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		event, ok, err := parseCableFrame(label, data)
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

func sendCableHeartbeats(ctx context.Context, conn *websocket.Conn, identifier string, interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			data, err := json.Marshal(map[string]string{"action": "heartbeat"})
			if err != nil {
				return
			}
			body, err := json.Marshal(map[string]string{
				"command":    "message",
				"identifier": identifier,
				"data":       string(data),
			})
			if err != nil {
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
				return
			}
		}
	}
}

func logSubscriberID() string {
	return fmt.Sprintf("sdk-%d", time.Now().UnixNano())
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
