package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

// Command is one JSONL input line.
type Command struct {
	Cmd     string `json:"cmd"`
	Client  string `json:"client,omitempty"`
	Filter  string `json:"filter,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Payload string `json:"payload,omitempty"`
	Retain  bool   `json:"retain,omitempty"`
	What    string `json:"what,omitempty"`
}

// Response is one JSONL output line.
type Response struct {
	OK            bool       `json:"ok"`
	Error         string     `json:"error,omitempty"`
	Replay        []Message  `json:"replay,omitempty"`
	Deliveries    []Delivery `json:"deliveries,omitempty"`
	Subscriptions any        `json:"subscriptions,omitempty"`
	Retained      any        `json:"retained,omitempty"`
}

func main() {
	statePath := flag.String("state", defaultStatePath(), "state file path")
	flag.Parse()

	broker, err := LoadBroker(*statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load state:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		enc.Encode(handle(broker, *statePath, line))
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read stdin:", err)
		os.Exit(1)
	}
}

func fail(msg string) Response {
	return Response{OK: false, Error: msg}
}

func handle(b *Broker, statePath string, line []byte) Response {
	var c Command
	if err := json.Unmarshal(line, &c); err != nil {
		return fail("invalid JSON: " + err.Error())
	}
	switch c.Cmd {
	case "subscribe":
		if c.Client == "" {
			return fail("subscribe requires a client id")
		}
		if err := validateFilter(c.Filter); err != nil {
			return fail(err.Error())
		}
		replay := b.Subscribe(c.Client, c.Filter)
		if err := SaveBroker(statePath, b); err != nil {
			return fail("save state: " + err.Error())
		}
		return Response{OK: true, Replay: replay}
	case "unsubscribe":
		if c.Client == "" {
			return fail("unsubscribe requires a client id")
		}
		if err := validateFilter(c.Filter); err != nil {
			return fail(err.Error())
		}
		if !b.Unsubscribe(c.Client, c.Filter) {
			return fail(fmt.Sprintf("client %q has no subscription %q", c.Client, c.Filter))
		}
		if err := SaveBroker(statePath, b); err != nil {
			return fail("save state: " + err.Error())
		}
		return Response{OK: true}
	case "publish":
		if err := validateTopicName(c.Topic); err != nil {
			return fail(err.Error())
		}
		deliveries := b.Publish(c.Topic, c.Payload, c.Retain)
		if c.Retain {
			if err := SaveBroker(statePath, b); err != nil {
				return fail("save state: " + err.Error())
			}
		}
		return Response{OK: true, Deliveries: deliveries}
	case "query":
		return query(b, c.What)
	default:
		return fail(fmt.Sprintf("unknown command %q", c.Cmd))
	}
}

func query(b *Broker, what string) Response {
	switch what {
	case "subscriptions":
		clients := make([]string, 0, len(b.Subs))
		for id := range b.Subs {
			clients = append(clients, id)
		}
		sort.Strings(clients)
		out := map[string][]string{}
		for _, id := range clients {
			filters := make([]string, 0, len(b.Subs[id]))
			for f := range b.Subs[id] {
				filters = append(filters, f)
			}
			sort.Strings(filters)
			out[id] = filters
		}
		return Response{OK: true, Subscriptions: out}
	case "retained":
		topics := make([]string, 0, len(b.Retained))
		for t := range b.Retained {
			topics = append(topics, t)
		}
		sort.Strings(topics)
		out := make([]Message, 0, len(topics))
		for _, t := range topics {
			out = append(out, Message{Topic: t, Payload: b.Retained[t]})
		}
		return Response{OK: true, Retained: out}
	default:
		return fail(`query requires what="subscriptions" or "retained"`)
	}
}
