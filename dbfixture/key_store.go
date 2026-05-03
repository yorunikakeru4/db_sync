package dbfixture

import (
	"context"
	"fmt"

	"github.com/uptrace/uptrace/pkg/json"
)

// KeyStore is the central registry of fixture key → PK JSON mappings.
// It replaces the per-loader pkByKey maps, the Fixture-level knownKeys,
// and the pendingKeys batch buffer.
//
// Keys are populated from two sources:
//   - LoadAll: reads persisted keys from the fixture_keys table (persistKeys mode)
//   - Set: called by loaders after Insert to register newly created PKs
//
// Mutations are tracked so they can be flushed to the database at the
// end of a Seed call.
type KeyStore struct {
	// keys[model][fixtureKey] = pkJSON
	keys map[string]map[string]json.RawMessage

	// dirty tracks keys added or modified since the last LoadAll/SaveDirty.
	dirty map[string]map[string]bool

	// deleted tracks keys removed since the last FlushDeleted.
	deleted map[string]map[string]bool
}

// Get returns the raw PK JSON for a (model, key) pair.
func (s *KeyStore) Get(model, key string) (json.RawMessage, bool) {
	if m := s.keys[model]; m != nil {
		pk, ok := m[key]
		return pk, ok
	}
	return nil, false
}

// Has reports whether a key exists for the given model.
func (s *KeyStore) Has(model, key string) bool {
	if m := s.keys[model]; m != nil {
		_, ok := m[key]
		return ok
	}
	return false
}

// Keys returns all fixture keys for a model.
// The returned map is the internal map — callers must not mutate it.
func (s *KeyStore) Keys(model string) map[string]json.RawMessage {
	return s.keys[model]
}

// Set stores a PK JSON for a (model, key) pair and marks it dirty.
func (s *KeyStore) Set(model, key string, pk json.RawMessage) {
	if s.keys == nil {
		s.keys = make(map[string]map[string]json.RawMessage)
	}
	m := s.keys[model]
	if m == nil {
		m = make(map[string]json.RawMessage)
		s.keys[model] = m
	}
	m[key] = pk

	if s.dirty == nil {
		s.dirty = make(map[string]map[string]bool)
	}
	dm := s.dirty[model]
	if dm == nil {
		dm = make(map[string]bool)
		s.dirty[model] = dm
	}
	dm[key] = true

	// If we re-add a key that was previously deleted in this cycle,
	// remove it from the deleted set.
	if s.deleted[model] != nil {
		delete(s.deleted[model], key)
	}
}

// Delete removes a key and marks it for deletion from the database.
// It returns an error if the key does not exist, which helps surface
// stale-key bugs (e.g. double deletes) rather than silently ignoring them.
func (s *KeyStore) Delete(model, key string) error {
	m := s.keys[model]
	if m == nil {
		return fmt.Errorf("dbfixture: KeyStore.Delete: unknown model %q", model)
	}
	if _, ok := m[key]; !ok {
		return fmt.Errorf("dbfixture: KeyStore.Delete: unknown key %q for model %q", key, model)
	}
	delete(m, key)

	// Remove from dirty — no need to save something we're deleting.
	if s.dirty[model] != nil {
		delete(s.dirty[model], key)
	}

	if s.deleted == nil {
		s.deleted = make(map[string]map[string]bool)
	}
	dm := s.deleted[model]
	if dm == nil {
		dm = make(map[string]bool)
		s.deleted[model] = dm
	}
	dm[key] = true
	return nil
}

// LoadAll reads all fixture keys from the database into the in-memory store.
// Existing in-memory keys are preserved (keys loaded here do not overwrite
// keys set during the current session).
func (s *KeyStore) LoadAll(ctx context.Context, repo *FixtureKeyRepo) error {
	allKeys, err := repo.ListAllKeys(ctx)
	if err != nil {
		return fmt.Errorf("load all fixture keys: %w", err)
	}

	if s.keys == nil {
		s.keys = make(map[string]map[string]json.RawMessage)
	}

	for _, fk := range allKeys {
		m := s.keys[fk.Model]
		if m == nil {
			m = make(map[string]json.RawMessage)
			s.keys[fk.Model] = m
		}
		// Only set if not already present (in-memory wins over DB).
		if _, exists := m[fk.Key]; !exists {
			m[fk.Key] = fk.PK
		}
	}

	return nil
}

// SaveDirty batch-upserts all dirty keys to the fixture_keys table
// and clears the dirty set.
func (s *KeyStore) SaveDirty(ctx context.Context, repo *FixtureKeyRepo) error {
	var batch []FixtureKey

	for model, dm := range s.dirty {
		for key := range dm {
			pk, ok := s.Get(model, key)
			if !ok {
				continue // deleted after being marked dirty
			}
			batch = append(batch, FixtureKey{
				Model: model,
				Key:   key,
				PK:    pk,
			})
		}
	}

	if len(batch) > 0 {
		if err := repo.SaveKeys(ctx, batch); err != nil {
			return err
		}
	}

	clear(s.dirty)
	return nil
}

// FlushDeleted batch-deletes all removed keys from the fixture_keys table
// and clears the deleted set.
func (s *KeyStore) FlushDeleted(ctx context.Context, repo *FixtureKeyRepo) error {
	for model, dm := range s.deleted {
		keys := make([]string, 0, len(dm))
		for key := range dm {
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			if err := repo.DeleteKeys(ctx, model, keys); err != nil {
				return err
			}
		}
	}

	clear(s.deleted)
	return nil
}
