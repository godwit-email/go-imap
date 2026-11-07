package imap

// StoreOptions contains options for the STORE command.
type StoreOptions struct {
	UnchangedSince uint64 // requires CONDSTORE
}

// StoreFlagsOp is a flag operation: set, add or delete.
type StoreFlagsOp int

const (
	StoreFlagsSet StoreFlagsOp = iota
	StoreFlagsAdd
	StoreFlagsDel
)

// StoreGmailLabels alters Gmail labels. Requires X-GM-EXT-1.
// Labels are UTF-8 mailbox names or system labels such as \Inbox.
// Add and delete preserve labels concurrently added by other clients.
type StoreGmailLabels struct {
	Op     StoreFlagsOp
	Silent bool
	Labels []string
}

// StoreFlags alters message flags.
type StoreFlags struct {
	Op     StoreFlagsOp
	Silent bool
	Flags  []Flag
}
