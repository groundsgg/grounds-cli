package ui

import (
	"fmt"
	"io"
	"strings"
)

// AskTypeName prompts for an exact-match string. Returns nil when the
// user types `expected`; an error otherwise. In non-TTY contexts the
// caller should pre-supply the typed value via flag.
func AskTypeName(in io.Reader, out io.Writer, prompt, expected string) error {
	fmt.Fprintf(out, "Type %s to confirm: ", prompt)
	var got string
	fmt.Fscanln(in, &got)
	got = strings.TrimSpace(got)
	if got != expected {
		return fmt.Errorf("input %q does not match", got)
	}
	return nil
}
