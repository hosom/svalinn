package main

import "testing"

func TestPassContainsUser(t *testing.T) {
	contains := passContainsUser("foo", "foobar9000")
	if !contains {
		t.Error("foobar9000 should contain foo")
	}

	contains = passContainsUser("foo", "bar123412341234")
	if contains {
		t.Error("bar123412341234 should not contain foo")
	}
}

func TestIsPwnd(t *testing.T) {
	pwnd := isPwnd("qwertyuio")

	if !pwnd {
		t.Error("Failed to validate the test for isPwnd.")
	}

	pwnd = isPwnd("dlfakjsdfouasdflajsdlfkjasdlfjaoisdfjalksdfnkajduhvyadofijalknkl")
	if pwnd {
		t.Error("Random password detected as pwnd.")
	}
}

func TestMetricEntropy(t *testing.T) {
	s := "abbcccdddd"

	if metricEntropy(s) != 0.18464393446710153 {
		t.Error("Failed to calculate correct entropy.", metricEntropy(s))
	}

	s = ""
	if metricEntropy(s) != 0 {
		t.Error("Failed to handle empty string.", metricEntropy(s))
	}
}
