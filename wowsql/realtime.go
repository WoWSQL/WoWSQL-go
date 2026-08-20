package WOWSQL

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BuildRealtimeWebSocketURL is the browser-safe auth URL: ?apikey= always.
func BuildRealtimeWebSocketURL(projectURL, apiKey string) string {
	origin := strings.TrimRight(projectURL, "/")
	ws := strings.Replace(origin, "https://", "wss://", 1)
	ws = strings.Replace(ws, "http://", "ws://", 1)
	return ws + "/realtime/v1/websocket?apikey=" + url.QueryEscape(apiKey)
}

// RealtimeChange is a postgres INSERT/UPDATE/DELETE fanout.
type RealtimeChange struct {
	Event   string                 `json:"event"`
	Schema  string                 `json:"schema"`
	Table   string                 `json:"table"`
	New     map[string]interface{} `json:"new,omitempty"`
	Old     map[string]interface{} `json:"old,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type realtimeSub struct {
	id       int
	schema   string
	table    string
	event    string
	callback func(RealtimeChange)
}

// RealtimeChannel is ephemeral broadcast + presence (no postgres trigger).
type RealtimeChannel struct {
	rt              *Realtime
	Name            string
	joined          bool
	state           map[string]interface{}
	broadcast       []func(map[string]interface{})
	presence        []func(map[string]interface{})
	postgres        []realtimeSub
	tracked         map[string]interface{}
	mu              sync.Mutex
}

// Realtime is the WowSQL websocket client (Python protocol, not Phoenix).
type Realtime struct {
	projectURL string
	apiKey     string
	conn       *websocket.Conn
	mu         sync.Mutex
	subs       []realtimeSub
	channels   map[string]*RealtimeChannel
	manual     bool
	attempts   int
	nextID     int
}

func newRealtime(projectURL, apiKey string) *Realtime {
	return &Realtime{
		projectURL: strings.TrimRight(projectURL, "/"),
		apiKey:     apiKey,
		channels:   map[string]*RealtimeChannel{},
	}
}

func (r *Realtime) URL() string {
	return BuildRealtimeWebSocketURL(r.projectURL, r.apiKey)
}

func (r *Realtime) Channel(name string) *RealtimeChannel {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch, ok := r.channels[name]; ok {
		return ch
	}
	ch := &RealtimeChannel{rt: r, Name: name, state: map[string]interface{}{}}
	r.channels[name] = ch
	return ch
}

func (r *Realtime) Subscribe(table string, cb func(RealtimeChange), schema, event string) func() {
	if schema == "" {
		schema = "public"
	}
	if event == "" {
		event = "*"
	}
	r.mu.Lock()
	r.nextID++
	sub := realtimeSub{id: r.nextID, schema: schema, table: table, event: event, callback: cb}
	r.subs = append(r.subs, sub)
	r.mu.Unlock()
	go func() {
		_ = r.ensureConnected()
		r.sendJSON(map[string]interface{}{"type": "subscribe", "schema": schema, "table": table, "event": event})
	}()
	return func() { r.unsubscribe(sub) }
}

func (r *Realtime) Close() {
	r.mu.Lock()
	r.manual = true
	c := r.conn
	r.conn = nil
	r.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
}

func (c *Client) Realtime() *Realtime {
	if c.realtime == nil {
		c.realtime = newRealtime(c.baseURL, c.apiKey)
	}
	return c.realtime
}

func (r *Realtime) sendJSON(msg map[string]interface{}) {
	r.mu.Lock()
	c := r.conn
	r.mu.Unlock()
	if c == nil {
		return
	}
	_ = c.WriteJSON(msg)
}

func (r *Realtime) ensureConnected() error {
	r.mu.Lock()
	if r.conn != nil {
		r.mu.Unlock()
		return nil
	}
	r.manual = false
	r.mu.Unlock()
	c, _, err := websocket.DefaultDialer.Dial(r.URL(), nil)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.conn = c
	r.attempts = 0
	subs := append([]realtimeSub(nil), r.subs...)
	chs := make([]*RealtimeChannel, 0, len(r.channels))
	for _, ch := range r.channels {
		chs = append(chs, ch)
	}
	r.mu.Unlock()
	for _, s := range subs {
		r.sendJSON(map[string]interface{}{"type": "subscribe", "schema": s.schema, "table": s.table, "event": s.event})
	}
	for _, ch := range chs {
		ch.rejoin()
	}
	go r.readLoop(c)
	return nil
}

func (r *Realtime) readLoop(c *websocket.Conn) {
	for {
		_, data, err := c.ReadMessage()
		if err != nil {
			r.mu.Lock()
			r.conn = nil
			manual := r.manual
			hasWork := len(r.subs) > 0 || len(r.channels) > 0
			r.mu.Unlock()
			if !manual && hasWork {
				delay := time.Duration(1<<uint(r.attempts)) * time.Second
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				r.attempts++
				time.Sleep(delay)
				_ = r.ensureConnected()
			}
			return
		}
		r.handle(data)
	}
}

func (r *Realtime) unsubscribe(tracked realtimeSub) {
	r.mu.Lock()
	next := r.subs[:0]
	for _, s := range r.subs {
		if s.id != tracked.id {
			next = append(next, s)
		}
	}
	r.subs = next
	r.mu.Unlock()
	r.sendJSON(map[string]interface{}{"type": "unsubscribe", "schema": tracked.schema, "table": tracked.table})
}

func (r *Realtime) handle(raw []byte) {
	var msg map[string]interface{}
	if json.Unmarshal(raw, &msg) != nil {
		return
	}
	if name, _ := msg["channel"].(string); name != "" {
		r.mu.Lock()
		ch := r.channels[name]
		r.mu.Unlock()
		if ch != nil {
			ch.handleServer(msg)
		}
	}
	if str(msg["type"]) != "broadcast" {
		return
	}
	if msg["channel"] != nil && msg["table"] == nil {
		return
	}
	nested, _ := msg["payload"].(map[string]interface{})
	if nested == nil {
		nested = map[string]interface{}{}
	}
	event := strings.ToUpper(first(str(msg["event"]), str(nested["type"])))
	schema := first(str(msg["schema"]), str(nested["schema"]), "public")
	table := first(str(msg["table"]), str(nested["table"]))
	if table == "" || (event != "INSERT" && event != "UPDATE" && event != "DELETE") {
		return
	}
	change := RealtimeChange{Event: event, Schema: schema, Table: table, Payload: nested}
	if n, ok := nested["new"].(map[string]interface{}); ok {
		change.New = n
	}
	if o, ok := nested["old"].(map[string]interface{}); ok {
		change.Old = o
	}
	r.mu.Lock()
	subs := append([]realtimeSub(nil), r.subs...)
	chs := make([]*RealtimeChannel, 0, len(r.channels))
	for _, ch := range r.channels {
		chs = append(chs, ch)
	}
	r.mu.Unlock()
	for _, s := range subs {
		if s.schema == schema && s.table == table && eventMatches(s.event, event) {
			s.callback(change)
		}
	}
	for _, ch := range chs {
		ch.handlePostgres(change)
	}
}

func (ch *RealtimeChannel) OnBroadcast(event string, cb func(map[string]interface{})) *RealtimeChannel {
	ch.broadcast = append(ch.broadcast, func(m map[string]interface{}) {
		if event == "*" || event == str(m["event"]) {
			cb(m)
		}
	})
	return ch
}

func (ch *RealtimeChannel) OnPresence(cb func(map[string]interface{})) *RealtimeChannel {
	ch.presence = append(ch.presence, cb)
	return ch
}

func (ch *RealtimeChannel) Subscribe(onStatus func(string)) *RealtimeChannel {
	go func() {
		_ = ch.rt.ensureConnected()
		ch.rt.sendJSON(map[string]interface{}{"type": "join", "channel": ch.Name})
		ch.joined = true
		if onStatus != nil {
			onStatus("SUBSCRIBED")
		}
	}()
	return ch
}

func (ch *RealtimeChannel) Send(event string, payload map[string]interface{}) {
	_ = ch.rt.ensureConnected()
	if !ch.joined {
		ch.rt.sendJSON(map[string]interface{}{"type": "join", "channel": ch.Name})
		ch.joined = true
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	ch.rt.sendJSON(map[string]interface{}{"type": "broadcast", "channel": ch.Name, "event": event, "payload": payload})
}

func (ch *RealtimeChannel) Track(payload map[string]interface{}) {
	ch.tracked = payload
	_ = ch.rt.ensureConnected()
	if !ch.joined {
		ch.rt.sendJSON(map[string]interface{}{"type": "join", "channel": ch.Name})
		ch.joined = true
	}
	ch.rt.sendJSON(map[string]interface{}{"type": "presence", "event": "track", "channel": ch.Name, "payload": payload})
}

func (ch *RealtimeChannel) PresenceState() map[string]interface{} {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	out := map[string]interface{}{}
	for k, v := range ch.state {
		out[k] = v
	}
	return out
}

func (ch *RealtimeChannel) Unsubscribe() {
	ch.rt.sendJSON(map[string]interface{}{"type": "leave", "channel": ch.Name})
	ch.joined = false
	ch.rt.mu.Lock()
	delete(ch.rt.channels, ch.Name)
	ch.rt.mu.Unlock()
}

func (ch *RealtimeChannel) rejoin() {
	ch.rt.sendJSON(map[string]interface{}{"type": "join", "channel": ch.Name})
	if ch.tracked != nil {
		ch.rt.sendJSON(map[string]interface{}{"type": "presence", "event": "track", "channel": ch.Name, "payload": ch.tracked})
	}
}

func (ch *RealtimeChannel) handleServer(msg map[string]interface{}) {
	switch str(msg["type"]) {
	case "joined":
		ch.joined = true
	case "presence":
		ev := str(msg["event"])
		ch.mu.Lock()
		if ev == "sync" {
			if st, ok := msg["state"].(map[string]interface{}); ok {
				ch.state = st
			}
		} else if ev == "join" {
			if k, ok := msg["key"].(string); ok {
				ch.state[k] = msg["payload"]
			}
		} else if ev == "leave" {
			if k, ok := msg["key"].(string); ok {
				delete(ch.state, k)
			}
		}
		ch.mu.Unlock()
		for _, cb := range ch.presence {
			cb(msg)
		}
	case "broadcast":
		payload, _ := msg["payload"].(map[string]interface{})
		if payload == nil {
			payload = map[string]interface{}{}
		}
		wrapped := map[string]interface{}{"event": str(msg["event"]), "payload": payload, "channel": ch.Name}
		for _, cb := range ch.broadcast {
			cb(wrapped)
		}
	}
}

func (ch *RealtimeChannel) handlePostgres(change RealtimeChange) {
	for _, s := range ch.postgres {
		if s.schema == change.Schema && s.table == change.Table && eventMatches(s.event, change.Event) {
			s.callback(change)
		}
	}
}

func eventMatches(filter, incoming string) bool {
	return filter == "*" || strings.EqualFold(filter, incoming)
}

func str(v interface{}) string {
	s, _ := v.(string)
	return s
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

