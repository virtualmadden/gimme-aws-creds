package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPick(t *testing.T) {
	var out bytes.Buffer
	SetIO(strings.NewReader("1\n"), &out)
	defer ResetIO()

	idx, err := Pick("Choose:", []string{"alpha", "beta", "gamma"})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if idx != 1 {
		t.Errorf("idx = %d, want 1", idx)
	}
}

func TestPickSingleOption(t *testing.T) {
	var out bytes.Buffer
	SetIO(strings.NewReader(""), &out)
	defer ResetIO()

	idx, err := Pick("Choose:", []string{"only"})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if idx != 0 {
		t.Errorf("idx = %d, want 0", idx)
	}
}

func TestPromptDefault(t *testing.T) {
	var out bytes.Buffer
	SetIO(strings.NewReader("\n"), &out)
	defer ResetIO()

	val, err := PromptDefault("Region", "us-east-1")
	if err != nil {
		t.Fatalf("PromptDefault: %v", err)
	}
	if val != "us-east-1" {
		t.Errorf("val = %q", val)
	}
}

func TestConfirm(t *testing.T) {
	var out bytes.Buffer
	SetIO(strings.NewReader("yes\n"), &out)
	defer ResetIO()

	ok, err := Confirm("Save")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}
