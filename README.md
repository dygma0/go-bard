# Go bard

> ATENTION: My go lang skill is very low, so this code is very bad. I'm sorry.

Unofficial Google bard API for Go.

## Cookies

- Visit https://bard.google.com/ (login with your account).
- F12 for console.
- Application → Cookies → __Secure-1PSID and __Secure-1PSIDTS and __Secure-1PSIDCC cookie value.

## Installation

```bash
go get github.com/dygma0/go-bard
```

## Usage

```go
func main() {
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
```
