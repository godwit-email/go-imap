package imapclient

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestReadBodyLenient(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		ok       bool
	}{
		{"valid", `(("text" "plain" ("charset" "utf-8") NIL NIL "7bit" 10 1 NIL NIL NIL NIL)("application" "pdf" NIL NIL {3}` + "\r\n" + `a"b "base64" 10 NIL ("attachment" ("filename" "a.pdf")) NIL NIL) "mixed" ("boundary" "x") NIL NIL NIL)`, true},
		// https://github.com/emersion/go-imap/issues/701: an empty multipart.
		{"empty multipart", `(("ALTERNATIVE" ("BOUNDARY" "e7") NIL NIL)("APPLICATION" "PDF" NIL NIL NIL "BASE64" 1476806 NIL ("ATTACHMENT" ("FILENAME" "a.pdf")) NIL) "MIXED" ("BOUNDARY" "7b") NIL NIL)`, false},
		// https://github.com/emersion/go-imap/issues/659: no SP between strings.
		{"adjacent strings", `(("text" "html" ("charset" "UTF-8") NIL NIL "quoted-printable" 1224 16)("application" "octet-stream" ("name" "a.pdf") NIL """" "base64" 65322) "MIXED")`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.in+" UID 7)\r\n")), imapwire.ConnSideClient)
			bs, err := readBodyLenient(dec, &Options{})
			if err != nil {
				t.Fatalf("readBodyLenient() = %v", err)
			}
			if mp, _ := bs.(*imap.BodyStructureMultiPart); (mp != nil) != tc.ok {
				t.Fatalf("readBodyLenient() = %#v, want parsed: %v", bs, tc.ok)
			} else if tc.ok && mp.Children[1].(*imap.BodyStructureSinglePart).Description != `a"b` {
				t.Fatalf("literal description = %q", mp.Children[1].(*imap.BodyStructureSinglePart).Description)
			}
			var uid uint32
			if !dec.ExpectSP() || !dec.ExpectAtom(new(string)) || !dec.ExpectSP() || !dec.ExpectNumber(&uid) || uid != 7 {
				t.Fatalf("decoder not left after the value: %v", dec.Err())
			}
		})
	}

	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(`("text" "plain"`)), imapwire.ConnSideClient)
	if _, err := readBodyLenient(dec, &Options{}); err == nil {
		t.Fatal("readBodyLenient() on an unterminated list: no error")
	}
}

// Tencent Exmail sends ("charset " " "utf-8"") for ("charset" "utf-8").
func TestReadBodyExmailCharset(t *testing.T) {
	in := `(("TEXT" "PLAIN" ("charset " " "utf-8"") NIL NIL "8BIT" 588 44 NIL NIL NIL)("TEXT" "HTML" ("CHARSET " " "gbk"") NIL NIL "8BIT" 2670 54 NIL NIL NIL) "ALTERNATIVE" ("BOUNDARY" "b1") NIL NIL)`
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(in)), imapwire.ConnSideClient)
	bs, err := readBodyLenient(dec, &Options{})
	mp, _ := bs.(*imap.BodyStructureMultiPart)
	if err != nil || mp == nil {
		t.Fatalf("readBodyLenient() = %#v, %v", bs, err)
	}
	for i, want := range []string{"utf-8", "gbk"} {
		if got := mp.Children[i].(*imap.BodyStructureSinglePart).Params; len(got) != 1 || got["charset"] != want {
			t.Errorf("part %d params = %v, want charset %s", i, got, want)
		}
	}

	// Any other atom in a parameter list is still an error.
	for _, in := range []string{`("text" "plain" ("charset" utf-8) NIL NIL "7bit" 1 1)`, `("text" "plain" ("name " " "a"") NIL NIL "7bit" 1 1)`} {
		dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(in)), imapwire.ConnSideClient)
		if bs, _ := readBodyLenient(dec, &Options{}); bs != nil {
			t.Errorf("%s: parsed as %#v", in, bs)
		}
	}
}

// Tencent Exmail answers a quoted HEADER.FIELDS name with an empty section.
func TestWriteHeaderFieldsAtoms(t *testing.T) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	writeFetchItemBodySection(imapwire.NewEncoder(w, imapwire.ConnSideClient), &imap.FetchItemBodySection{
		Specifier: imap.PartSpecifierHeader, HeaderFields: []string{"SUBJECT", "X-A]", "X B"}, Peek: true})
	w.Flush()
	if want := `BODY.PEEK[HEADER.FIELDS (SUBJECT "X-A]" "X B")]`; b.String() != want {
		t.Errorf("got %s, want %s", b.String(), want)
	}
}
