package sse

import (
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	EventDialogStarted       EventType = "dialog.started"
	EventDialogClarification EventType = "dialog.clarification"
	EventRouteDetermined     EventType = "route.determined"
	EventDialogCompleted     EventType = "dialog.completed"
	EventSpecChanged         EventType = "spec.changed"
	EventTestStarted         EventType = "test.started"
	EventTestCompleted       EventType = "test.completed"
	EventNodeAdded           EventType = "canvas.node.added"
	EventConnectionAdded     EventType = "canvas.connection.added"
	EventThreadSwitched      EventType = "thread.switched"
)

type SSEEvent struct {
	EventId   string      `json:"eventId"`
	Event     EventType   `json:"event"`
	ThreadId  string     `json:"threadId"`
	Payload   interface{}`json:"payload"`
	Timestamp string      `json:"timestamp"`
}

type Emitter struct {
	mu      sync.RWMutex
	clients map[string]chan SSEEvent
	nextId  int
}

func NewEmitter() *Emitter {
	return &Emitter{
		clients: make(map[string]chan SSEEvent),
		nextId:  1,
	}
}

func (e *Emitter) Register(clientId string) chan SSEEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan SSEEvent, 256)
	e.clients[clientId] = ch
	return ch
}

func (e *Emitter) Unregister(clientId string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch, ok := e.clients[clientId]; ok {
		close(ch)
	}
	delete(e.clients, clientId)
}

func (e *Emitter) Emit(event SSEEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if event.EventId == "" {
		event.EventId = fmt.Sprintf("%d", e.nextId)
		e.nextId++
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, ch := range e.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (e *Emitter) EmitJSON(eventType EventType, threadId string, payload interface{}) {
	event := SSEEvent{
		Event:    eventType,
		ThreadId: threadId,
		Payload:  payload,
	}
	e.Emit(event)
}

func (e *Emitter) BroadcastJSON(eventType EventType, payload interface{}) {
	e.EmitJSON(eventType, "global", payload)
}

// Stream sends events to a client channel directly (for HTTP SSE handler)
func (e *Emitter) Stream(clientId string, ch chan SSEEvent) {
	e.mu.Lock()
	e.clients[clientId] = ch
	e.mu.Unlock()
	// Keep alive - channel closed by Unregister
	<-ch
}
