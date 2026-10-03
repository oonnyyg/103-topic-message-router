package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

type command struct {
	Cmd     string `json:"cmd"`
	Client  string `json:"client,omitempty"`
	Filter  string `json:"filter,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Payload string `json:"payload,omitempty"`
	Retain  bool   `json:"retain,omitempty"`
	What    string `json:"what,omitempty"`
}

type response struct {
	OK            bool                `json:"ok"`
	Error         string              `json:"error,omitempty"`
	Replay        []Message           `json:"replay,omitempty"`
	Delivered     []string            `json:"delivered,omitempty"`
	Subscriptions map[string][]string `json:"subscriptions,omitempty"`
	Retained      []Message           `json:"retained,omitempty"`
}

func main() {
	statePath := flag.String("state", "router-state.json", "path to the state file")
	flag.Parse()

	broker, err := LoadBroker(*statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load state:", err)
		os.Exit(1)
	}
	if err := run(broker, *statePath, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(b *Broker, statePath string, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(out)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var cmd command
		if err := json.Unmarshal(raw, &cmd); err != nil {
			enc.Encode(response{OK: false, Error: fmt.Sprintf("line %d: invalid JSON: %v", line, err)})
			continue
		}
		enc.Encode(handle(b, statePath, cmd))
	}
	return scanner.Err()
}

func handle(b *Broker, statePath string, cmd command) response {
	switch cmd.Cmd {
	case "subscribe":
		replay, err := b.Subscribe(cmd.Client, cmd.Filter)
		if err != nil {
			return response{OK: false, Error: err.Error()}
		}
		if err := SaveBroker(statePath, b); err != nil {
			return response{OK: false, Error: "save state: " + err.Error()}
		}
		return response{OK: true, Replay: replay}
	case "unsubscribe":
		if err := b.Unsubscribe(cmd.Client, cmd.Filter); err != nil {
			return response{OK: false, Error: err.Error()}
		}
		if err := SaveBroker(statePath, b); err != nil {
			return response{OK: false, Error: "save state: " + err.Error()}
		}
		return response{OK: true}
	case "publish":
		delivered, err := b.Publish(cmd.Topic, cmd.Payload, cmd.Retain)
		if err != nil {
			return response{OK: false, Error: err.Error()}
		}
		if err := SaveBroker(statePath, b); err != nil {
			return response{OK: false, Error: "save state: " + err.Error()}
		}
		return response{OK: true, Delivered: delivered}
	case "query":
		switch cmd.What {
		case "subscriptions":
			return response{OK: true, Subscriptions: b.Subscriptions()}
		case "retained":
			return response{OK: true, Retained: b.RetainedMessages()}
		default:
			return response{OK: false, Error: `query "what" must be "subscriptions" or "retained"`}
		}
	default:
		return response{OK: false, Error: fmt.Sprintf("unknown command %q", cmd.Cmd)}
	}
}
