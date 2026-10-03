package main

import "sort"

// Message is a routed message; Payload is carried verbatim.
type Message struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// Broker holds subscriptions and retained messages.
type Broker struct {
	// Subs maps client ID to the set of filters (deduplicated).
	Subs map[string]map[string]bool `json:"subs"`
	// Retained maps topic to payload.
	Retained map[string]string `json:"retained"`
}

func NewBroker() *Broker {
	return &Broker{
		Subs:     map[string]map[string]bool{},
		Retained: map[string]string{},
	}
}

// Subscribe adds filter for client. Re-subscribing an existing filter does
// not increase the count but still replays retained messages.
// Returns retained messages matching the filter, sorted by topic.
func (b *Broker) Subscribe(client, filter string) ([]Message, error) {
	if client == "" {
		return nil, errEmptyClient
	}
	if err := ValidateFilter(filter); err != nil {
		return nil, err
	}
	if b.Subs[client] == nil {
		b.Subs[client] = map[string]bool{}
	}
	b.Subs[client][filter] = true
	var replay []Message
	for topic, payload := range b.Retained {
		if Match(filter, topic) {
			replay = append(replay, Message{Topic: topic, Payload: payload})
		}
	}
	sort.Slice(replay, func(i, j int) bool { return replay[i].Topic < replay[j].Topic })
	return replay, nil
}

// Unsubscribe removes filter for client.
func (b *Broker) Unsubscribe(client, filter string) error {
	if client == "" {
		return errEmptyClient
	}
	if err := ValidateFilter(filter); err != nil {
		return err
	}
	filters, ok := b.Subs[client]
	if !ok || !filters[filter] {
		return errNotSubscribed
	}
	delete(filters, filter)
	if len(filters) == 0 {
		delete(b.Subs, client)
	}
	return nil
}

// Publish routes a message. Each matching client receives it at most once;
// results are sorted by client ID. Retained publishes replace or clear the
// stored message but are still delivered to current subscribers.
func (b *Broker) Publish(topic, payload string, retain bool) ([]string, error) {
	if err := ValidateTopic(topic); err != nil {
		return nil, err
	}
	var clients []string
	for client, filters := range b.Subs {
		for filter := range filters {
			if Match(filter, topic) {
				clients = append(clients, client)
				break
			}
		}
	}
	sort.Strings(clients)
	if retain {
		if payload == "" {
			delete(b.Retained, topic)
		} else {
			b.Retained[topic] = payload
		}
	}
	return clients, nil
}

// Subscriptions returns a sorted snapshot of client->filters.
func (b *Broker) Subscriptions() map[string][]string {
	out := map[string][]string{}
	for client, filters := range b.Subs {
		list := make([]string, 0, len(filters))
		for f := range filters {
			list = append(list, f)
		}
		sort.Strings(list)
		out[client] = list
	}
	return out
}

// RetainedMessages returns retained messages sorted by topic.
func (b *Broker) RetainedMessages() []Message {
	out := make([]Message, 0, len(b.Retained))
	for topic, payload := range b.Retained {
		out = append(out, Message{Topic: topic, Payload: payload})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Topic < out[j].Topic })
	return out
}
