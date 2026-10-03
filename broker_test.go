package main

import (
	"path/filepath"
	"testing"
)

func TestPublishDeduplicatesOverlappingSubscriptions(t *testing.T) {
	b := NewBroker()
	b.Subscribe("c1", "a/#")
	b.Subscribe("c1", "a/b")
	b.Subscribe("c2", "a/+")
	got := b.Publish("a/b", "hello", false)
	if len(got) != 2 {
		t.Fatalf("expected 2 deliveries, got %d: %+v", len(got), got)
	}
	// sorted by client id
	if got[0].ClientID != "c1" || got[1].ClientID != "c2" {
		t.Fatalf("deliveries not sorted by client: %+v", got)
	}
	if got[0].Payload != "hello" || got[0].Topic != "a/b" {
		t.Fatalf("message content altered: %+v", got[0])
	}
}

func TestDuplicateSubscribeKeepsCountButReplays(t *testing.T) {
	b := NewBroker()
	b.Publish("x/1", "v1", true)
	b.Publish("x/2", "v2", true)
	b.Subscribe("c", "x/+")
	replay := b.Subscribe("c", "x/+") // duplicate
	if n := b.SubscriptionCount("c"); n != 1 {
		t.Fatalf("expected 1 subscription, got %d", n)
	}
	if len(replay) != 2 || replay[0].Topic != "x/1" || replay[1].Topic != "x/2" {
		t.Fatalf("replay not sorted by topic or incomplete: %+v", replay)
	}
}

func TestRetainedReplaceAndClear(t *testing.T) {
	b := NewBroker()
	b.Subscribe("c", "t/#")
	b.Publish("t/a", "old", true)
	b.Publish("t/a", "new", true) // replace
	if b.Retained["t/a"] != "new" {
		t.Fatalf("retained not replaced: %q", b.Retained["t/a"])
	}
	// empty retained publish clears but still delivers
	d := b.Publish("t/a", "", true)
	if _, ok := b.Retained["t/a"]; ok {
		t.Fatal("empty retained publish should clear the retained message")
	}
	if len(d) != 1 || d[0].ClientID != "c" {
		t.Fatalf("clearing publish should still deliver: %+v", d)
	}
	// plain publish does not touch retained
	b.Publish("t/b", "v", true)
	b.Publish("t/b", "changed", false)
	if b.Retained["t/b"] != "v" {
		t.Fatalf("plain publish must not change retained: %q", b.Retained["t/b"])
	}
}

func TestUnsubscribe(t *testing.T) {
	b := NewBroker()
	b.Subscribe("c", "a/#")
	if !b.Unsubscribe("c", "a/#") {
		t.Fatal("unsubscribe should succeed")
	}
	if b.Unsubscribe("c", "a/#") {
		t.Fatal("second unsubscribe should report absence")
	}
	if got := b.Publish("a/b", "x", false); len(got) != 0 {
		t.Fatalf("no deliveries expected after unsubscribe: %+v", got)
	}
}

func TestRestartRestoresSubscriptionsAndRetained(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	b := NewBroker()
	b.Subscribe("c1", "sensors/#")
	b.Publish("sensors/temp", "21.5", true)
	if err := SaveBroker(path, b); err != nil {
		t.Fatal(err)
	}

	// simulate restart
	b2, err := LoadBroker(path)
	if err != nil {
		t.Fatal(err)
	}
	// retained replay on a new subscription after restart
	replay := b2.Subscribe("c2", "sensors/+")
	if len(replay) != 1 || replay[0].Payload != "21.5" {
		t.Fatalf("replay after restart wrong: %+v", replay)
	}
	// restored subscription still receives publishes (c2 also matches)
	d := b2.Publish("sensors/hum", "40", false)
	if len(d) != 2 || d[0].ClientID != "c1" || d[1].ClientID != "c2" {
		t.Fatalf("restored subscription not routed: %+v", d)
	}
}
