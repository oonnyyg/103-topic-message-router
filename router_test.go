package main

import "testing"

func TestMatchFilter(t *testing.T) {
	cases := []struct {
		filter, topic string
		want          bool
	}{
		{"a/b", "a/b", true},
		{"a/b", "a/B", false},  // case-sensitive
		{"a//b", "a//b", true}, // empty levels preserved
		{"a//b", "a/x/b", false},
		{"a/+", "a/b", true},
		{"a/+", "a/b/c", false}, // '+' matches exactly one level
		{"a/+", "a", false},
		{"a/+", "a/", true}, // '+' matches an empty level
		{"a/#", "a", true},  // '#' matches zero levels
		{"a/#", "a/b/c", true},
		{"#", "a/b/c", true},
		{"#", "$sys/status", false}, // first-level wildcard skips '$'
		{"+/status", "$sys/status", false},
		{"$sys/#", "$sys/status", true},
	}
	for _, c := range cases {
		if got := matchFilter(c.filter, c.topic); got != c.want {
			t.Errorf("matchFilter(%q,%q)=%v want %v", c.filter, c.topic, got, c.want)
		}
	}
}

func TestValidateFilter(t *testing.T) {
	valid := []string{"a/b", "+", "a/+", "#", "a/#", "a//+"}
	for _, f := range valid {
		if err := validateFilter(f); err != nil {
			t.Errorf("validateFilter(%q) unexpected error: %v", f, err)
		}
	}
	invalid := []string{"", "a+", "a/b+", "a#", "a/#/b", "#/a", "a/#/#"}
	for _, f := range invalid {
		if err := validateFilter(f); err == nil {
			t.Errorf("validateFilter(%q) expected error", f)
		}
	}
}

func TestValidateTopicName(t *testing.T) {
	for _, topic := range []string{"", "a/+", "a/#", "#"} {
		if err := validateTopicName(topic); err == nil {
			t.Errorf("validateTopicName(%q) expected error", topic)
		}
	}
	if err := validateTopicName("a//B"); err != nil {
		t.Errorf("validateTopicName valid topic: %v", err)
	}
}
