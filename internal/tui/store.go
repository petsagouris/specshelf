package tui

import "github.com/petsagouris/specshelf/internal/vault"

// Store is everything the session asks of the vault: the document list with one
// warning per document that could not be read, and, for the detail view, one
// document's bytes. The session performs every read and — from 5.4 — every
// mutation through this interface, so the package never touches a vault file.
// internal/vault satisfies it in production; tests supply a counting fake.
type Store interface {
	List() ([]vault.IndexRecord, []vault.Warning, error)
	Find(ref string) (vault.Found, error)
}

// VaultStore is the production Store. Every call forwards to internal/vault, the
// only package allowed to touch a vault file; List uses Query so the session
// shows documents in the same order as the list command.
type VaultStore struct{ Root string }

// List returns every document in list order.
func (v VaultStore) List() ([]vault.IndexRecord, []vault.Warning, error) {
	return vault.Query(v.Root, vault.Filters{})
}

// Find returns one document's bytes by reference.
func (v VaultStore) Find(ref string) (vault.Found, error) {
	return vault.Find(v.Root, ref)
}
