package main

import "testing"

func TestValidateFilter(t *testing.T) {
	valid := []string{"a/b", "a/+", "+/+", "#", "a/#", "a//b", "/+", "a//+"}
	for _, f := range valid {
		if err := ValidateFilter(f); err != nil {
			t.Errorf("filter %q should be valid: %v", f, err)
		}
	}
	invalid := []string{"", "a+", "a/b+", "+a", "a#", "a/#/b", "#/a", "a/b#"}
	for _, f := range invalid {
		if err := ValidateFilter(f); err == nil {
			t.Errorf("filter %q should be invalid", f)
		}
	}
}

func TestValidateTopic(t *testing.T) {
	for _, topic := range []string{"a/b", "a//b", "$SYS/info"} {
		if err := ValidateTopic(topic); err != nil {
			t.Errorf("topic %q should be valid: %v", topic, err)
		}
	}
	for _, topic := range []string{"", "a/+", "a/#", "#"} {
		if err := ValidateTopic(topic); err == nil {
			t.Errorf("topic %q should be invalid", topic)
		}
	}
}

func TestMatchWildcardBoundaries(t *testing.T) {
	cases := []struct {
		filter, topic string
		want          bool
	}{
		{"a/+", "a/b", true},
		{"a/+", "a/b/c", false},  // '+' matches exactly one level
		{"a/+", "a", false},
		{"a/+", "a/", true},      // empty level preserved and matched
		{"a/#", "a", true},       // '#' matches zero levels
		{"a/#", "a/b/c", true},   // '#' matches many levels
		{"#", "a/b", true},
		{"#", "$SYS/x", false},   // first-level wildcard vs '$' topic
		{"+/b", "$SYS/b", false},
		{"$SYS/#", "$SYS/x", true},
		{"a/B", "a/b", false},    // case sensitive
		{"a//b", "a//b", true},   // empty levels significant
		{"a//b", "a/b", false},
		{"a/b", "a/b/", false},
	}
	for _, c := range cases {
		if got := Match(c.filter, c.topic); got != c.want {
			t.Errorf("Match(%q, %q) = %v, want %v", c.filter, c.topic, got, c.want)
		}
	}
}
