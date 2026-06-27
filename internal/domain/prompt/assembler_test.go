package prompt

import (
	"testing"
)

func TestAssemblePrompt_Empty(t *testing.T) {
	result := AssemblePrompt(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestAssemblePrompt_Single(t *testing.T) {
	items := []AssemblyItem{
		{Content: "masterpiece", SortOrder: 0},
	}
	result := AssemblePrompt(items)
	if result != "masterpiece" {
		t.Errorf("expected 'masterpiece', got %q", result)
	}
}

func TestAssemblePrompt_SortedByOrder(t *testing.T) {
	items := []AssemblyItem{
		{Content: "third", SortOrder: 2},
		{Content: "first", SortOrder: 0},
		{Content: "second", SortOrder: 1},
	}
	result := AssemblePrompt(items)
	expected := "first, second, third"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestAssemblePrompt_StableSort(t *testing.T) {
	// Same sort order → stable (first seen stays first)
	items := []AssemblyItem{
		{Content: "A", SortOrder: 1},
		{Content: "B", SortOrder: 1},
		{Content: "C", SortOrder: 0},
	}
	result := AssemblePrompt(items)
	expected := "C, A, B"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestAssemblePrompt_CommaSeparated(t *testing.T) {
	items := []AssemblyItem{
		{Content: "sunset beach", SortOrder: 0},
		{Content: "digital art", SortOrder: 1},
	}
	result := AssemblePrompt(items)
	expected := "sunset beach, digital art"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
