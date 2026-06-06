package main

import "testing"

func TestNormalizeAnswer(t *testing.T) {
	if got := normalizeAnswer(" Twelve! "); got != "twelve" {
		t.Fatalf("normalizeAnswer() = %q, want twelve", got)
	}
	if got := normalizeAnswer("12.0"); got != "120" {
		t.Fatalf("normalizeAnswer() = %q, want 120", got)
	}
}
