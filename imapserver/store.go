package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SessionGmailLabels is a session which supports STORE X-GM-LABELS.
type SessionGmailLabels interface {
	Session
	StoreGmailLabels(w *FetchWriter, numSet imap.NumSet, labels *imap.StoreGmailLabels) error
}

func (c *Conn) handleStoreGmailLabels(dec *imapwire.Decoder, set imap.NumSet, item string) error {
	store, err := readStoreGmailLabels(dec, item)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	sess, ok := c.session.(SessionGmailLabels)
	if !ok {
		return newClientBugError("STORE X-GM-LABELS is not supported")
	}
	return sess.StoreGmailLabels(&FetchWriter{conn: c}, set, store)
}

func readStoreGmailLabels(dec *imapwire.Decoder, item string) (*imap.StoreGmailLabels, error) {
	item = strings.ToUpper(item)
	store := &imap.StoreGmailLabels{Silent: strings.HasSuffix(item, ".SILENT")}
	item = strings.TrimSuffix(item, ".SILENT")
	if strings.HasPrefix(item, "+") {
		store.Op = imap.StoreFlagsAdd
		item = item[1:]
	} else if strings.HasPrefix(item, "-") {
		store.Op = imap.StoreFlagsDel
		item = item[1:]
	}
	if item != "X-GM-LABELS" {
		return nil, newClientBugError("invalid STORE item")
	}
	if err := dec.ExpectList(func() error {
		var label string
		if dec.Special('\\') {
			if !dec.ExpectAtom(&label) {
				return dec.Err()
			}
			label = "\\" + label
		} else if !dec.ExpectMailbox(&label) {
			return dec.Err()
		}
		store.Labels = append(store.Labels, label)
		return nil
	}); err != nil {
		return nil, err
	}
	if !dec.ExpectCRLF() {
		return nil, dec.Err()
	}
	return store, nil
}

func (c *Conn) handleStore(dec *imapwire.Decoder, numKind NumKind) error {
	var (
		numSet imap.NumSet
		item   string
	)
	if !dec.ExpectSP() || !dec.ExpectNumSet(numKind.wire(), &numSet) || !dec.ExpectSP() || !dec.ExpectAtom(&item) || !dec.ExpectSP() {
		return dec.Err()
	}
	if strings.Contains(strings.ToUpper(item), "X-GM-LABELS") {
		return c.handleStoreGmailLabels(dec, numSet, item)
	}
	var flags []imap.Flag
	isList, err := dec.List(func() error {
		flag, err := internal.ExpectFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, flag)
		return nil
	})
	if err != nil {
		return err
	} else if !isList {
		for {
			flag, err := internal.ExpectFlag(dec)
			if err != nil {
				return err
			}
			flags = append(flags, flag)

			if !dec.SP() {
				break
			}
		}
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	item = strings.ToUpper(item)
	silent := strings.HasSuffix(item, ".SILENT")
	item = strings.TrimSuffix(item, ".SILENT")

	var op imap.StoreFlagsOp
	switch {
	case strings.HasPrefix(item, "+"):
		op = imap.StoreFlagsAdd
		item = strings.TrimPrefix(item, "+")
	case strings.HasPrefix(item, "-"):
		op = imap.StoreFlagsDel
		item = strings.TrimPrefix(item, "-")
	default:
		op = imap.StoreFlagsSet
	}

	if item != "FLAGS" {
		return newClientBugError("STORE can only change FLAGS")
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	w := &FetchWriter{conn: c}
	options := imap.StoreOptions{}
	return c.session.Store(w, numSet, &imap.StoreFlags{
		Op:     op,
		Silent: silent,
		Flags:  flags,
	}, &options)
}
