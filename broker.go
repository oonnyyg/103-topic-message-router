package main

import "sort"

// Message is a routed payload; Payload is delivered verbatim.
type Message struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// Delivery is a message addressed to one client.
type Delivery struct {
	ClientID string `json:"client"`
	Topic    string `json:"topic"`
	Payload  string `json:"payload"`
}

// Broker holds subscriptions and retained messages.
type Broker struct {
	// Subs maps clientID -> filter set (deduplicated).
	Subs map[string]map[string]bool `json:"subs"`
	// Retained maps topic -> payload.
	Retained map[string]string `json:"retained"`
}

func NewBroker() *Broker {
	return &Broker{
		Subs:     map[string]map[string]bool{},
		Retained: map[string]string{},
	}
}

// Subscribe adds filter for clientID. Re-subscribing an existing filter
// does not change the count but still replays retained messages.
// Returns the retained replay for this filter, sorted by topic.
func (b *Broker) Subscribe(clientID, filter string) []Message {
	if b.Subs[clientID] == nil {
		b.Subs[clientID] = map[string]bool{}
	}
	b.Subs[clientID][filter] = true
	return b.replay(filter)
}

// Unsubscribe removes filter for clientID. Returns false if absent.
func (b *Broker) Unsubscribe(clientID, filter string) bool {
	set, ok := b.Subs[clientID]
	if !ok || !set[filter] {
		return false
	}
	delete(set, filter)
	if len(set) == 0 {
		delete(b.Subs, clientID)
	}
	return true
}

// Publish routes a message. Retained publishes replace the retained copy;
// an empty-payload retained publish clears it but is still delivered.
// Deliveries are sorted by client ID, one per client.
func (b *Broker) Publish(topic, payload string, retain bool) []Delivery {
	if retain {
		if payload == "" {
			delete(b.Retained, topic)
		} else {
			b.Retained[topic] = payload
		}
	}
	clients := make([]string, 0, len(b.Subs))
	for id, filters := range b.Subs {
		for f := range filters {
			if matchFilter(f, topic) {
				clients = append(clients, id)
				break // one delivery per client per publish
			}
		}
	}
	sort.Strings(clients)
	out := make([]Delivery, 0, len(clients))
	for _, id := range clients {
		out = append(out, Delivery{ClientID: id, Topic: topic, Payload: payload})
	}
	return out
}

// replay returns retained messages matching filter, sorted by topic.
func (b *Broker) replay(filter string) []Message {
	topics := make([]string, 0)
	for topic := range b.Retained {
		if matchFilter(filter, topic) {
			topics = append(topics, topic)
		}
	}
	sort.Strings(topics)
	out := make([]Message, 0, len(topics))
	for _, topic := range topics {
		out = append(out, Message{Topic: topic, Payload: b.Retained[topic]})
	}
	return out
}

// SubscriptionCount returns the number of distinct filters for a client.
func (b *Broker) SubscriptionCount(clientID string) int {
	return len(b.Subs[clientID])
}
