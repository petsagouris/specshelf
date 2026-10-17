package tui

import "github.com/petsagouris/specshelf/internal/vault"

// Store is everything the session asks of the vault: the document list with one
// warning per document that could not be read, one document's bytes for the
// detail view, and the edit handoff. The session performs every read and
// mutation through this interface, so the package never touches a vault file.
// internal/vault satisfies it in production; tests supply a counting fake.
type Store interface {
	List() ([]vault.IndexRecord, []vault.Warning, error)
	Find(ref string) (vault.Found, error)
	Create(in vault.NewDocument) (string, error)
	Edit(ref string) (string, error)
	SetStatus(ref, status string) (string, error)
	Remove(ref string) (string, error)
}

// VaultStore is the production Store. Every call forwards to internal/vault, the
// only package allowed to touch a vault file; List uses Query so the session
// shows documents in the same order as the list command. Editor is the
// vault.Editor the session hands a document to; internal/cli supplies the same
// implementation its `edit` command uses.
type VaultStore struct {
	Root   string
	Editor vault.Editor
}

// List returns every document in list order.
func (v VaultStore) List() ([]vault.IndexRecord, []vault.Warning, error) {
	return vault.Query(v.Root, vault.Filters{})
}

// Find returns one document's bytes by reference.
func (v VaultStore) Find(ref string) (vault.Found, error) {
	return vault.Find(v.Root, ref)
}

// Create composes a document from the vault's template, exactly as the `new`
// and `decision` commands do.
func (v VaultStore) Create(in vault.NewDocument) (string, error) {
	return vault.Create(v.Root, in)
}

// Edit hands one document to the editor, which stamps updated and refreshes the
// index only when the editor left different bytes behind.
func (v VaultStore) Edit(ref string) (string, error) {
	return vault.Edit(v.Root, ref, v.Editor)
}

// SetStatus sets one document's status and refreshes the index.
func (v VaultStore) SetStatus(ref, status string) (string, error) {
	return vault.SetStatus(v.Root, ref, status)
}

// Remove deletes one document and refreshes the index.
func (v VaultStore) Remove(ref string) (string, error) {
	return vault.Remove(v.Root, ref)
}
