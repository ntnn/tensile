package tensile

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

// IdentityMaxPairs bounds identity key/value pairs so [Identity]
// stays comparable and usable as a map key.
//
// [Identity] is pass-by-value and used as a map key, costing
// 16+32*IdentityMaxPairs bytes per copy on 64-bit platforms.
// With 8 pairs that is 272 B, including empty slots, not including
// what filled slots contain.
// Hence a small-ish number to prevent sprawl - both the number and
// contents of identity fields should be kept small.
const IdentityMaxPairs = 8

// kv is a single key/value pair of an [Identity].
type kv struct {
	key, value string
}

// Identity is a deterministic, human-readable node identifier.
// Identity is comparable and usable as a map key.
//
// The string form is `kind[key1="value1" key2="value2"]`.
// It was chosen to be both readable and parseable.
// See [IdentityRegex] for a regex to parse identities from text.
// For JSON logs it supports [slog.LogValuer] natively.
type Identity struct {
	kind  string
	pairs [IdentityMaxPairs]kv
}

// IdentityRegex is a regex to get the string form of a full [Identity]
// out of e.g. log statements.
const IdentityRegex = `\w+\[(\w+="(\\.|[^"\\])*" ?)*\]`

// AsIdentity builds an [Identity] from a kind and key/value pairs.
// Pair order is preserved.
// Panics on an odd number of key/value elements, empty keys, or more
// than [IdentityMaxPairs] pairs.
func AsIdentity(kind string, kv ...string) Identity {
	// a pair is one key and one value
	const pairSize = 2

	if len(kv)%pairSize != 0 {
		panic(fmt.Sprintf("AsIdentity: odd number of key/value elements: %d", len(kv)))
	}
	if len(kv)/pairSize > IdentityMaxPairs {
		panic(fmt.Sprintf("AsIdentity: more than %d pairs: %d", IdentityMaxPairs, len(kv)/pairSize))
	}

	id := Identity{kind: kind}
	i := 0
	for pair := range slices.Chunk(kv, pairSize) {
		if pair[0] == "" {
			panic("AsIdentity: empty key")
		}
		id.pairs[i].key = pair[0]
		id.pairs[i].value = pair[1]
		i++
	}
	return id
}

// Kind returns the kind.
func (id Identity) Kind() string {
	return id.kind
}

// Identity implements [Identifier], returning itself.
func (id Identity) Identity() Identity {
	return id
}

// String implements [fmt.Stringer], returning the identity in the form `kind[key1="value1" key2="value2"]`.
func (id Identity) String() string {
	if id.pairs[0].key == "" {
		return id.kind
	}

	var b strings.Builder
	b.WriteString(id.kind)
	b.WriteByte('[')
	for i, pair := range id.pairs {
		if pair.key == "" {
			break
		}
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s=%q", pair.key, pair.value)
	}
	b.WriteByte(']')
	return b.String()
}

// LogValue implements [slog.LogValuer].
func (id Identity) LogValue() slog.Value {
	attrs := []slog.Attr{slog.String("kind", id.kind)}
	for _, pair := range id.pairs {
		if pair.key == "" {
			break
		}
		attrs = append(attrs, slog.String(pair.key, pair.value))
	}
	return slog.GroupValue(attrs...)
}
