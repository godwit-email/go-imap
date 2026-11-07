package imapserver

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestReadStoreGmailLabels(t *testing.T) {
	for _, tc := range []struct {
		item string
		op   imap.StoreFlagsOp
	}{{"+X-GM-LABELS.SILENT", imap.StoreFlagsAdd}, {"-X-GM-LABELS.SILENT", imap.StoreFlagsDel}} {
		dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(`(\Inbox "Muy Importante" "&XeVPXA-")`+"\r\n")), imapwire.ConnSideServer)
		got, err := readStoreGmailLabels(dec, tc.item)
		if err != nil {
			t.Fatal(err)
		}
		if got.Op != tc.op || !got.Silent || !reflect.DeepEqual(got.Labels, []string{`\Inbox`, "Muy Importante", "工作"}) {
			t.Fatalf("labels = %#v", got)
		}
	}
}
