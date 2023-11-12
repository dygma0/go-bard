package gobard

import (
	"testing"
)

func TestBard_Ask(t *testing.T) {
	b := NewBard(
		Secure1PSID(""),
		Secure1PSIDCC(""),
		Secure1PSIDTS(""),
	)
	prompt := "What is the meaning of life?"

	answer, err := b.Ask(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Log(answer)
}
