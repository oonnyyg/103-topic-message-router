package main

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateFilter checks a subscription filter.
// "+" must occupy a whole level; "#" must be a whole level and the last one.
func ValidateFilter(filter string) error {
	if filter == "" {
		return errors.New("filter must not be empty")
	}
	levels := strings.Split(filter, "/")
	for i, lv := range levels {
		if strings.Contains(lv, "+") && lv != "+" {
			return fmt.Errorf("invalid filter %q: '+' must occupy an entire level", filter)
		}
		if strings.Contains(lv, "#") {
			if lv != "#" {
				return fmt.Errorf("invalid filter %q: '#' must occupy an entire level", filter)
			}
			if i != len(levels)-1 {
				return fmt.Errorf("invalid filter %q: '#' must be the last level", filter)
			}
		}
	}
	return nil
}

// ValidateTopic checks a publish topic: non-empty, no wildcards.
func ValidateTopic(topic string) error {
	if topic == "" {
		return errors.New("topic must not be empty")
	}
	if strings.ContainsAny(topic, "+#") {
		return fmt.Errorf("invalid topic %q: wildcards not allowed in publish topic", topic)
	}
	return nil
}

// Match reports whether filter matches topic. A wildcard in the first level
// does not match topics whose first level starts with '$'.
func Match(filter, topic string) bool {
	f := strings.Split(filter, "/")
	t := strings.Split(topic, "/")
	dollar := strings.HasPrefix(topic, "$")
	for i, lv := range f {
		if lv == "#" {
			if i == 0 && dollar {
				return false
			}
			return true
		}
		if i >= len(t) {
			return false
		}
		if lv == "+" {
			if i == 0 && dollar {
				return false
			}
			continue
		}
		if lv != t[i] {
			return false
		}
	}
	return len(f) == len(t)
}
