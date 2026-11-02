package imapwire_test

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func newTestDecoder(s string) *imapwire.Decoder {
	return imapwire.NewDecoder(bufio.NewReader(strings.NewReader(s)), imapwire.ConnSideServer)
}

func expectDecoderExpectError(t *testing.T, dec *imapwire.Decoder) {
	t.Helper()
	err := dec.Err()
	if err == nil {
		t.Fatal("decoder has no error")
	}
	var decErr *imapwire.DecoderExpectError
	if !errors.As(err, &decErr) {
		t.Fatalf("decoder error is %T (%v), want *DecoderExpectError", err, err)
	}
}

func TestExpectNumSetInvalid(t *testing.T) {
	dec := newTestDecoder("abc\r\n")

	var numSet imap.NumSet
	if dec.ExpectNumSet(imapwire.NumKindSeq, &numSet) {
		t.Fatal("ExpectNumSet accepted an invalid sequence-set")
	}
	expectDecoderExpectError(t, dec)
}

func TestExpectMailboxInvalidUTF7(t *testing.T) {
	dec := newTestDecoder("&\r\n")

	var name string
	if dec.ExpectMailbox(&name) {
		t.Fatal("ExpectMailbox accepted a name that is not modified UTF-7")
	}
	expectDecoderExpectError(t, dec)
}

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
		dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.in)), imapwire.ConnSideClient)
		lit, _, ok := dec.LiteralReader()
		if !ok {
			t.Fatalf("%q: no literal: %v", tc.in, dec.Err())
		}
		io.Copy(io.Discard, lit)
		dec.Special(')') // a peek, as a list does before SP, keeps the exception
		if got := dec.SP(); got != tc.sp {
			t.Errorf("%q: SP() = %v, want %v", tc.in, got, tc.sp)
		}
	}

	// Only right after the literal: a string read since then ends the exception.
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader("{0}\r\n \"a\"B")), imapwire.ConnSideClient)
	lit, _, _ := dec.LiteralReader()
	io.Copy(io.Discard, lit)
	var s string
	if !dec.SP() || !dec.String(&s) || dec.SP() {
		t.Error("SP() accepted a missing SP after a string")
	}
}
