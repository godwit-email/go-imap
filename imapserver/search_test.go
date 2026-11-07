package imapserver

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestReadSearchGmailMsgID(t *testing.T) {
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader("X-GM-MSGID 18000000000000000001\r\n")), imapwire.ConnSideServer)
	var criteria imap.SearchCriteria
	if err := readSearchKey(&criteria, dec); err != nil {
		t.Fatal(err)
	}
	if len(criteria.GmailMsgID) != 1 || criteria.GmailMsgID[0] != 18000000000000000001 {
		t.Fatalf("criteria = %#v", criteria)
	}
	var combined imap.SearchCriteria
	combined.And(&criteria)
	if len(combined.GmailMsgID) != 1 || combined.GmailMsgID[0] != 18000000000000000001 {
		t.Fatalf("And lost identity: %#v", combined)
	}
}

func TestReadSearchKeyUnknownKey(t *testing.T) {
	// An unquoted argument with a space leaves "test" as the next key.
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader("test\r\n")), imapwire.ConnSideServer)

	var criteria imap.SearchCriteria
	err := readSearchKey(&criteria, dec)
	if err == nil {
		t.Fatal("readSearchKey succeeded on an unknown search key")
	}

	var imapErr *imap.Error
	if !errors.As(err, &imapErr) {
		t.Fatalf("readSearchKey returned %T (%v), want *imap.Error", err, err)
	}
	if imapErr.Type != imap.StatusResponseTypeBad {
		t.Errorf("error type is %q, want %q", imapErr.Type, imap.StatusResponseTypeBad)
	}
	if imapErr.Code != imap.ResponseCodeClientBug {
		t.Errorf("error code is %q, want %q", imapErr.Code, imap.ResponseCodeClientBug)
	}
}
