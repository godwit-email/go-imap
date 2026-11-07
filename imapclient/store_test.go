package imapclient_test

import (
	"bufio"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestStoreGmailLabels(t *testing.T) {
	for _, tc := range []struct {
		op     imap.StoreFlagsOp
		prefix string
	}{{imap.StoreFlagsAdd, "+"}, {imap.StoreFlagsDel, "-"}} {
		t.Run(tc.prefix, func(t *testing.T) {
			cc, sc := net.Pipe()
			defer sc.Close()
			reqCh := make(chan string, 1)
			go func() {
				io.WriteString(sc, "* OK ready\r\n")
				req, _ := bufio.NewReader(sc).ReadString('\n')
				reqCh <- req
				tag, _, _ := strings.Cut(req, " ")
				io.WriteString(sc, tag+" OK done\r\n")
			}()
			c := imapclient.New(cc, nil)
			defer c.Close()
			err := c.StoreGmailLabels(imap.UIDSetNum(7), &imap.StoreGmailLabels{Op: tc.op, Silent: true, Labels: []string{`\Inbox`, "Muy Importante", "工作"}}).Close()
			if err != nil {
				t.Fatal(err)
			}
			req := <-reqCh
			want := ` UID STORE 7 ` + tc.prefix + `X-GM-LABELS.SILENT ("\\Inbox" "Muy Importante" "&XeVPXA-")` + "\r\n"
			if !strings.HasSuffix(req, want) {
				t.Fatalf("request = %q, want suffix %q", req, want)
			}
		})
	}
}

func TestSearchGmailMsgID(t *testing.T) {
	cc, sc := net.Pipe()
	defer sc.Close()
	reqCh := make(chan string, 1)
	go func() {
		io.WriteString(sc, "* OK ready\r\n")
		r := bufio.NewReader(sc)
		req, _ := r.ReadString('\n')
		// UIDSearch may request capabilities before encoding SEARCH.
		if strings.Contains(req, "CAPABILITY") {
			tag, _, _ := strings.Cut(req, " ")
			io.WriteString(sc, "* CAPABILITY IMAP4rev1 X-GM-EXT-1\r\n"+tag+" OK done\r\n")
			req, _ = r.ReadString('\n')
		}
		reqCh <- req
		tag, _, _ := strings.Cut(req, " ")
		io.WriteString(sc, "* SEARCH 7\r\n"+tag+" OK done\r\n")
	}()
	c := imapclient.New(cc, nil)
	defer c.Close()
	got, err := c.UIDSearch(&imap.SearchCriteria{GmailMsgID: []uint64{18000000000000000001}}, nil).Wait()
	if err != nil {
		t.Fatal(err)
	}
	if req := <-reqCh; !strings.Contains(req, "UID SEARCH X-GM-MSGID 18000000000000000001\r\n") {
		t.Fatalf("request = %q", req)
	}
	if !reflect.DeepEqual(got.AllUIDs(), []imap.UID{7}) {
		t.Fatalf("result = %#v", got)
	}
}

func TestStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqSet := imap.SeqSetNum(1)
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	msgs, err := client.Store(seqSet, &storeFlags, nil).Collect()
	if err != nil {
		t.Fatalf("Store().Collect() = %v", err)
	} else if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %v, want %v", len(msgs), 1)
	}
	msg := msgs[0]
	if msg.SeqNum != 1 {
		t.Errorf("msg.SeqNum = %v, want %v", msg.SeqNum, 1)
	}

	found := false
	for _, f := range msg.Flags {
		if f == imap.FlagDeleted {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("msg.Flags is missing deleted flag: %v", msg.Flags)
	}
}
