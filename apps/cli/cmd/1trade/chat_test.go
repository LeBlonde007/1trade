package main

import "testing"

// TestSplitSlash checks the REPL slash-command parser: lowercased command, trimmed trailing argument,
// and the no-argument case.
func TestSplitSlash(t *testing.T) {
	cases := []struct{ in, cmd, rest string }{
		{"/model llama-3.1-70b", "model", "llama-3.1-70b"},
		{"/MODEL  gpt-5 ", "model", "gpt-5"},
		{"/exit", "exit", ""},
		{"/save  chat.txt", "save", "chat.txt"},
		{"/help", "help", ""},
	}
	for _, c := range cases {
		cmd, rest := splitSlash(c.in)
		if cmd != c.cmd || rest != c.rest {
			t.Errorf("splitSlash(%q) = (%q,%q), want (%q,%q)", c.in, cmd, rest, c.cmd, c.rest)
		}
	}
}

// TestParseFloat tolerates fixed-point strings and bad input (→ 0, cosmetic only).
func TestParseFloat(t *testing.T) {
	if parseFloat("982.500000") != 982.5 {
		t.Errorf("parseFloat fixed-point = %v", parseFloat("982.500000"))
	}
	if parseFloat("") != 0 || parseFloat("x") != 0 {
		t.Error("bad input should parse to 0")
	}
}
