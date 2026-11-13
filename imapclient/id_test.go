package imapclient

import (
	"bufio"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// readID must leave no error on the decoder for a valid response: the decoder
// keeps its first error, so a stale one would be what a later, unrelated
// failure on the connection reports (godwit-email/godwit#749: an Outlook ID
// list, then a dropped IDLE reported as `expected atom, got "("`).
func TestReadIDLeavesNoDecoderError(t *testing.T) {
	for _, tc := range []struct {
		name, in, wantName string
	}{
		{"list", ` ("name" "Microsoft.Exchange.Imap4.Imap4Server" "version" "15.20")`, "Microsoft.Exchange.Imap4.Imap4Server"},
		{"nil", ` NIL`, ""},
		{"lower-case nil", ` nil`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.in+"\r\n")), imapwire.ConnSideClient)
			data, err := (&Client{}).readID(dec)
			if err != nil {
				t.Fatalf("readID() = %v", err)
			}
			if data.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", data.Name, tc.wantName)
			}
			if err := dec.Err(); err != nil {
				t.Errorf("decoder error left behind: %v", err)
			}
			if !dec.ExpectCRLF() {
				t.Errorf("not at the line's end: %v", dec.Err())
			}
		})
	}
}

func TestReadIDRejectsOtherAtom(t *testing.T) {
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(" FOO\r\n")), imapwire.ConnSideClient)
	if _, err := (&Client{}).readID(dec); err == nil {
		t.Fatal("readID() accepted an atom other than NIL")
	}
}
