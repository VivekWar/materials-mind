package services

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizeEmbeddingInput_CollapsesWhitespace(t *testing.T) {
	got := normalizeEmbeddingInput("  titanium   alloy\n\tfor  drones  ")
	want := "titanium alloy for drones"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeEmbeddingInput_TruncatesToMaxRunes(t *testing.T) {
	long := strings.Repeat("a ", maxEmbeddingInputRune)
	got := normalizeEmbeddingInput(long)
	if utf8.RuneCountInString(got) != maxEmbeddingInputRune {
		t.Fatalf("expected truncation to %d runes, got %d", maxEmbeddingInputRune, utf8.RuneCountInString(got))
	}
}

func TestNormalizeEmbeddingInput_EmptyStaysEmpty(t *testing.T) {
	if got := normalizeEmbeddingInput("   \n\t  "); got != "" {
		t.Fatalf("expected empty string for whitespace-only input, got %q", got)
	}
}
