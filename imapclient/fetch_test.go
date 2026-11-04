package imapclient_test

import (
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestFetch(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqSet := imap.SeqSetNum(1)
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("failed to fetch first message: %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	msg := messages[0]
	if len(msg.BodySection) != 1 {
		t.Fatalf("len(msg.BodySection) = %v, want 1", len(msg.BodySection))
	}
	b := msg.FindBodySection(bodySection)
	if b == nil {
		t.Fatalf("FindBodySection() = nil")
	}
	body := strings.ReplaceAll(string(b), "\r\n", "\n")
	if body != simpleRawMessage {
		t.Errorf("body mismatch: got \n%v\n but want \n%v", body, simpleRawMessage)
	}
}

func TestFetchPartialOffsetEcho(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	const offset = int64(1)<<32 + 5

	bodySection := &imap.FetchItemBodySection{
		Partial: &imap.SectionPartial{Offset: offset, Size: 10},
	}
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(messages) != 1 || len(messages[0].BodySection) != 1 {
		t.Fatalf("got %v messages", len(messages))
	}

	got := messages[0].BodySection[0].Section.Partial
	if got == nil || got.Offset != offset {
		t.Errorf("echoed partial offset = %v, want %v", got, offset)
	}
	if messages[0].FindBodySection(bodySection) == nil {
		t.Errorf("FindBodySection() = nil, want the section the client asked for")
	}
}

// Alibaba Mail echoes a HEADER.FIELDS list cut down and reordered; the section
// still holds the headers it sent, and an exact echo is preferred.
func TestFindBodySectionReducedHeaderFields(t *testing.T) {
	asked := &imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader, Peek: true,
		HeaderFields: []string{"MESSAGE-ID", "DATE", "SUBJECT", "FROM", "LIST-ID", "DKIM-SIGNATURE"}}
	hdr := func(part []int, b string, names ...string) imapclient.FetchBodySectionBuffer {
		return imapclient.FetchBodySectionBuffer{Section: &imap.FetchItemBodySection{
			Specifier: imap.PartSpecifierHeader, Part: part, HeaderFields: names}, Bytes: []byte(b)}
	}
	buf := &imapclient.FetchMessageBuffer{BodySection: []imapclient.FetchBodySectionBuffer{
		{Section: &imap.FetchItemBodySection{Specifier: imap.PartSpecifierText}, Bytes: []byte("body")},
		hdr(nil, "reduced", "From", "SUBJECT", "date", "MESSAGE-ID"),
	}}
	if got := string(buf.FindBodySection(asked)); got != "reduced" {
		t.Fatalf("reduced echo: got %q", got)
	}
	buf.BodySection = append(buf.BodySection, hdr(nil, "exact", asked.HeaderFields...))
	if got := string(buf.FindBodySection(asked)); got != "exact" {
		t.Fatalf("exact echo not preferred: got %q", got)
	}

	for _, s := range []imapclient.FetchBodySectionBuffer{
		hdr(nil, "x", "FROM", "TO"), // a name not asked for
		hdr([]int{2}, "x", "FROM"),  // another part
		{Section: &imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader, HeaderFieldsNot: []string{"TO"}}, Bytes: []byte("x")},
		{Section: &imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader}, Bytes: []byte("x")},
	} {
		buf := &imapclient.FetchMessageBuffer{BodySection: []imapclient.FetchBodySectionBuffer{s}}
		if got := buf.FindBodySection(asked); got != nil {
			t.Errorf("%+v matched: %q", s.Section, got)
		}
	}
}
