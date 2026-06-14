package storage

import "strconv"

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
// Components are LENGTH-PREFIXED before joining (see encodeCacheTuple). A bare
// delimiter join — including the NUL ('\x00') join a previous implementation
// relied on — is NOT injection-proof: it merely assumes the delimiter never
// appears in any field. sourceText is free-form user content and the
// provider/model fields are externally supplied, so that assumption is unsafe —
// e.g. (provider="p\x00q", model="m") and (provider="p", model="q\x00m") both
// NUL-join to "...p\x00q\x00m\x00x", colliding two DISTINCT tuples onto one hash
// and serving the WRONG cached translation. Length-prefixing makes the encoding
// unambiguous for ANY byte content (a field's bytes can never be mistaken for a
// delimiter), so distinct tuples never serialise to the same string. sha256 is
// collision-resistant, so distinct serialisations map to distinct hashes.
func cacheLookupHash(sourceText, sourceLanguage, targetLanguage, provider, model string) string {
	return hashString(encodeCacheTuple(sourceLanguage, targetLanguage, provider, model, sourceText))
}

// encodeCacheTuple serialises the cache-key components into an unambiguous,
// injection-proof string by prefixing each field with its byte length and a ':'.
// Because the length is fixed before the field bytes are read, no field's content
// (any byte, including delimiters or NUL) can shift another field's boundary, so
// distinct tuples can never produce the same encoding. Shared by both the
// SQLite/PostgreSQL lookup_hash and the Redis cache-key suffix so all three
// backends agree on the key for a tuple.
func encodeCacheTuple(fields ...string) string {
	// Pre-size: each field contributes its bytes + a small length-prefix header.
	n := 0
	for _, f := range fields {
		n += len(f) + 12
	}
	b := make([]byte, 0, n)
	for _, f := range fields {
		b = strconv.AppendInt(b, int64(len(f)), 10)
		b = append(b, ':')
		b = append(b, f...)
	}
	return string(b)
}
