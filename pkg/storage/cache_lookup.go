package storage

import "strings"

// cacheLookupHash returns the deterministic UNIQUE key for a translation-cache
// lookup tuple (source_text, source_language, target_language, provider, model).
//
// WHY a hash column instead of a multi-column UNIQUE index on the raw tuple:
// source_text can be very large (full paragraphs / chapters). A UNIQUE B-tree
// index that includes source_text is impractical on both backends — PostgreSQL
// rejects index entries larger than ~2704 bytes ("index row size ... exceeds
// btree version 4 maximum"), and a full-text key is wasteful on SQLite too. A
// single sha256 column gives a bounded, fixed-width (64 hex) key that the UNIQUE
// index covers on BOTH backends identically.
//
// Components are NUL-joined (matching the Redis cache-key convention in this
// package): '\x00' cannot appear in normal text, so the join is injection-proof
// — no two distinct tuples can serialise to the same string (e.g. ("a","b")
// vs ("a\x00b","")). sha256 is collision-resistant, so distinct tuples map to
// distinct hashes.
func cacheLookupHash(sourceText, sourceLanguage, targetLanguage, provider, model string) string {
	return hashString(strings.Join(
		[]string{sourceLanguage, targetLanguage, provider, model, sourceText},
		"\x00",
	))
}
