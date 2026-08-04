package models

import (
	"database/sql/driver"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStringArray_Value verifies the encoded PostgreSQL array-literal format
// byte-for-byte. This is the exact on-disk format of notifications.filter_tags
// (written historically by pq.StringArray), so it must not drift.
func TestStringArray_Value(t *testing.T) {
	cases := []struct {
		name string
		in   StringArray
		want driver.Value // nil means SQL NULL
	}{
		{"nil is NULL", nil, nil},
		{"empty is {}", StringArray{}, "{}"},
		{"single bare", StringArray{"rss"}, "{rss}"},
		{"multiple bare", StringArray{"rss", "starred", "priority"}, "{rss,starred,priority}"},
		{"element with space is quoted", StringArray{"has space"}, `{"has space"}`},
		{"element with comma is quoted", StringArray{"a,b"}, `{"a,b"}`},
		{"element with quote is escaped", StringArray{`quote"here`}, `{"quote\"here"}`},
		{"element with backslash is escaped", StringArray{`back\slash`}, `{"back\\slash"}`},
		{"element with brace is quoted", StringArray{"a}b"}, `{"a}b"}`},
		{"NULL token is quoted", StringArray{"NULL"}, `{"NULL"}`},
		{"mixed", StringArray{"rss", "my folder", "feed,2"}, `{rss,"my folder","feed,2"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.Value()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestStringArray_Scan verifies decoding of array literals, including the exact
// "{a,b}" shape that existing production rows (written by pq.StringArray and
// the Phase-6b ETL) hold today.
func TestStringArray_Scan(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want StringArray
	}{
		{"nil -> nil", nil, nil},
		{"empty braces", "{}", StringArray{}},
		{"prod-shaped bare", "{rss,starred,priority}", StringArray{"rss", "starred", "priority"}},
		{"single", "{rss}", StringArray{"rss"}},
		{"byte slice form", []byte("{rss,starred}"), StringArray{"rss", "starred"}},
		{"quoted space", `{"has space"}`, StringArray{"has space"}},
		{"quoted comma", `{"a,b"}`, StringArray{"a,b"}},
		{"escaped quote", `{"quote\"here"}`, StringArray{`quote"here`}},
		{"escaped backslash", `{"back\\slash"}`, StringArray{`back\slash`}},
		{"quoted brace", `{"a}b"}`, StringArray{"a}b"}},
		{"mixed", `{rss,"my folder","feed,2"}`, StringArray{"rss", "my folder", "feed,2"}},
		{"tolerates spaces around elements", `{ a , b }`, StringArray{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got StringArray
			require.NoError(t, got.Scan(tc.in))
			if tc.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, []string(tc.want), []string(got))
			}
		})
	}
}

// TestStringArray_ScanErrors covers the failure paths.
func TestStringArray_ScanErrors(t *testing.T) {
	var a StringArray
	assert.Error(t, a.Scan(42))                       // unsupported type
	assert.Error(t, a.Scan("not an array"))           // missing braces
	assert.Error(t, a.Scan(`{"unterminated`))         // unterminated quoted element
}

// TestStringArray_RoundTrip confirms Value -> Scan returns the input for a wide
// range of element shapes, including ones that require quoting/escaping.
func TestStringArray_RoundTrip(t *testing.T) {
	inputs := [][]string{
		{},
		{"rss"},
		{"rss", "starred", "priority"},
		{"has space", "no-space"},
		{"comma,inside", "plain"},
		{`quote"mark`, `back\slash`},
		{"{brace}", "}trailing", "{leading"},
		{"UPPER", "Mixed", "null", "NULL"}, // NULL token survives when it's not the only token
		{" Technologies ", "tabs\tand\nnewlines"},
	}
	for _, in := range inputs {
		src := StringArray(in)
		v, err := src.Value()
		require.NoError(t, err)
		var dst StringArray
		require.NoError(t, dst.Scan(v))
		assert.Equal(t, in, []string(dst), "round-trip for %v", in)
	}
}

// TestStringArray_FuzzRoundTrip throws many short random element sets through
// Value -> Scan to catch any quoting/escaping corner the explicit cases miss.
func TestStringArray_FuzzRoundTrip(t *testing.T) {
	const alphabet = `abcXYZ ,{}"\\ 0 \t\n`
	const n = 2000
	rng := newDeterministicRng(1)
	for i := 0; i < n; i++ {
		count := int(rng.next() % 4) // 0..3 elements
		in := make([]string, count)
		for j := range in {
			length := int(rng.next() % 5)
			var b []byte
			for k := 0; k < length; k++ {
				b = append(b, alphabet[rng.next()%uint64(len(alphabet))])
			}
			in[j] = string(b)
		}
		v, err := StringArray(in).Value()
		require.NoError(t, err)
		var dst StringArray
		require.NoError(t, dst.Scan(v))
		if !reflect.DeepEqual(in, []string(dst)) {
			t.Fatalf("round-trip mismatch:\n  in=%q\n  encoded=%v\n  out=%q", in, v, []string(dst))
		}
	}
	// Touch sort so the import is used even if the loop body changes; keeps
	// random element order stable per seed.
	_ = sort.Strings
}

// deterministicRng is a tiny xorshift generator so the fuzz test is reproducible
// (no dependency on math/rand seeding rules across Go versions).
type deterministicRng struct{ state uint64 }

func newDeterministicRng(seed uint64) *deterministicRng {
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15
	}
	return &deterministicRng{state: seed}
}

func (r *deterministicRng) next() uint64 {
	x := r.state
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	r.state = x
	return x
}
