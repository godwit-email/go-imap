package imapwire

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

// Strato / Open-Xchange send "{0}\r\nUID 42": no SP after an empty literal.
func TestDecoderSPAfterEmptyLiteral(t *testing.T) {
	for _, tc := range []struct {
		in string
		sp bool
	}{
		{"{0}\r\nUID 42", true},
		{"{0}\r\n UID 42", true},
		{"{0}\r\n)", false},
		{"{1}\r\nxUID 42", false},
	} {
		dec := NewDecoder(bufio.NewReader(strings.NewReader(tc.in)), ConnSideClient)
		lit, _, ok := dec.LiteralReader()
		if !ok {
			t.Fatalf("%q: no literal: %v", tc.in, dec.Err())
		}
		io.Copy(io.Discard, lit)
		if got := dec.SP(); got != tc.sp {
			t.Errorf("%q: SP() = %v, want %v", tc.in, got, tc.sp)
		}
	}

	// Only right after the literal: a string read since then ends the exception.
	dec := NewDecoder(bufio.NewReader(strings.NewReader("{0}\r\n \"a\"B")), ConnSideClient)
	lit, _, _ := dec.LiteralReader()
	io.Copy(io.Discard, lit)
	var s string
	if !dec.SP() || !dec.String(&s) || dec.SP() {
		t.Error("SP() accepted a missing SP after a string")
	}
}
