package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestOverlappingSubscriptionsDeliverOnce(t *testing.T) {
	b := NewBroker()
	mustSub(t, b, "c1", "a/+")
	mustSub(t, b, "c1", "a/#")
	mustSub(t, b, "c1", "a/b")
	mustSub(t, b, "c2", "#")

	got, err := b.Publish("a/b", "hello", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c1", "c2"} // sorted, c1 only once
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("delivered = %v, want %v", got, want)
	}
	if n := len(b.Subs["c1"]); n != 3 {
		t.Fatalf("c1 filter count = %d, want 3", n)
	}
}

func TestDuplicateSubscribeNoGrowthButReplays(t *testing.T) {
	b := NewBroker()
	if _, err := b.Publish("a/b", "v1", true); err != nil {
		t.Fatal(err)
	}
	replay1, err := b.Subscribe("c1", "a/#")
	if err != nil {
		t.Fatal(err)
	}
	replay2, err := b.Subscribe("c1", "a/#")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Subs["c1"]) != 1 {
		t.Fatalf("duplicate subscribe grew filter set: %v", b.Subs["c1"])
	}
	if !reflect.DeepEqual(replay1, replay2) || len(replay2) != 1 {
		t.Fatalf("replay mismatch: %v vs %v", replay1, replay2)
	}
}

func TestRetainedLifecycle(t *testing.T) {
	b := NewBroker()
	mustSub(t, b, "c1", "t/#")

	// Retained publish stores and delivers.
	if _, err := b.Publish("t/x", "v1", true); err != nil {
		t.Fatal(err)
	}
	// Normal publish does not change retained content.
	if _, err := b.Publish("t/x", "v2", false); err != nil {
		t.Fatal(err)
	}
	if b.Retained["t/x"] != "v1" {
		t.Fatalf("retained = %q, want v1", b.Retained["t/x"])
	}
	// Retained publish replaces.
	if _, err := b.Publish("t/x", "v3", true); err != nil {
		t.Fatal(err)
	}
	// Empty-payload retained publish clears but still delivers.
	delivered, err := b.Publish("t/x", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Retained["t/x"]; ok {
		t.Fatal("empty retained publish should clear the message")
	}
	if !reflect.DeepEqual(delivered, []string{"c1"}) {
		t.Fatalf("clearing publish should still deliver, got %v", delivered)
	}
	// New subscriber sees nothing now.
	replay, err := b.Subscribe("c2", "t/#")
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 0 {
		t.Fatalf("unexpected replay after clear: %v", replay)
	}
}

func TestReplaySortedByTopic(t *testing.T) {
	b := NewBroker()
	for _, topic := range []string{"m/b", "m/a", "m/c"} {
		if _, err := b.Publish(topic, "p-"+topic, true); err != nil {
			t.Fatal(err)
		}
	}
	replay, err := b.Subscribe("c1", "m/#")
	if err != nil {
		t.Fatal(err)
	}
	var topics []string
	for _, m := range replay {
		topics = append(topics, m.Topic)
	}
	if !reflect.DeepEqual(topics, []string{"m/a", "m/b", "m/c"}) {
		t.Fatalf("replay order = %v", topics)
	}
}

func TestRestartReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	b := NewBroker()
	mustSub(t, b, "c1", "s/+")
	if _, err := b.Publish("s/1", "keep-me", true); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Publish("s/2", "temp", false); err != nil {
		t.Fatal(err)
	}
	if err := SaveBroker(path, b); err != nil {
		t.Fatal(err)
	}

	// Simulate restart.
	b2, err := LoadBroker(path)
	if err != nil {
		t.Fatal(err)
	}
	// Subscription survives: publish routes without re-subscribing.
	delivered, err := b2.Publish("s/3", "after-restart", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(delivered, []string{"c1"}) {
		t.Fatalf("delivered after restart = %v", delivered)
	}
	// New subscriber replays only the retained message, payload verbatim.
	replay, err := b2.Subscribe("c2", "s/#")
	if err != nil {
		t.Fatal(err)
	}
	want := []Message{{Topic: "s/1", Payload: "keep-me"}}
	if !reflect.DeepEqual(replay, want) {
		t.Fatalf("replay after restart = %v, want %v", replay, want)
	}
}

func TestInvalidCommandsKeepState(t *testing.T) {
	b := NewBroker()
	mustSub(t, b, "c1", "a/#")
	before := b.Subscriptions()

	if _, err := b.Subscribe("c1", "a/#/bad"); err == nil {
		t.Fatal("expected error for bad filter")
	}
	if _, err := b.Publish("a/+", "x", false); err == nil {
		t.Fatal("expected error for wildcard topic")
	}
	if _, err := b.Publish("", "x", false); err == nil {
		t.Fatal("expected error for empty topic")
	}
	if err := b.Unsubscribe("c1", "never/subscribed"); err == nil {
		t.Fatal("expected error for unknown subscription")
	}
	if !reflect.DeepEqual(b.Subscriptions(), before) {
		t.Fatalf("state changed after invalid commands: %v", b.Subscriptions())
	}
}

func mustSub(t *testing.T, b *Broker, client, filter string) {
	t.Helper()
	if _, err := b.Subscribe(client, filter); err != nil {
		t.Fatalf("subscribe %s %s: %v", client, filter, err)
	}
}
