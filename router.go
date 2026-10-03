package main

import (
	"errors"
	"fmt"
	"strings"
)

// splitLevels splits a topic or filter into slash-separated levels,
// preserving empty levels. Matching is case-sensitive throughout.
func splitLevels(s string) []string {
	return strings.Split(s, "/")
}

// validateTopicName checks a publish topic: non-empty, no wildcards.
func validateTopicName(topic string) error {
	if topic == "" {
		return errors.New("publish topic must not be empty")
	}
	if strings.ContainsAny(topic, "+#") {
		return fmt.Errorf("publish topic %q must not contain wildcards", topic)
	}
	return nil
}

// validateFilter checks a subscription filter: '+' must occupy a whole
// level, '#' must occupy a whole level and be the last level.
func validateFilter(filter string) error {
	if filter == "" {
		return errors.New("filter must not be empty")
	}
	levels := splitLevels(filter)
	for i, lv := range levels {
		if strings.Contains(lv, "+") && lv != "+" {
			return fmt.Errorf("filter %q: '+' must occupy an entire level", filter)
		}
		if strings.Contains(lv, "#") {
			if lv != "#" {
				return fmt.Errorf("filter %q: '#' must occupy an entire level", filter)
			}
			if i != len(levels)-1 {
				return fmt.Errorf("filter %q: '#' must be the last level", filter)
			}
		}
	}
	return nil
}

// matchFilter reports whether filter matches topic. A wildcard in the
// first level does not match topics starting with '$'.
func matchFilter(filter, topic string) bool {
	f := splitLevels(filter)
	t := splitLevels(topic)
	if len(t) > 0 && strings.HasPrefix(t[0], "$") &&
		len(f) > 0 && (f[0] == "+" || f[0] == "#") {
		return false
	}
	for i, lv := range f {
		switch lv {
		case "#":
			return true // matches zero or more remaining levels
		case "+":
			if i >= len(t) {
				return false
			}
		default:
			if i >= len(t) || t[i] != lv {
				return false
			}
		}
	}
	return len(f) == len(t)
}
