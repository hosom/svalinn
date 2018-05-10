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
