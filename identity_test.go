package tensile_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsIdentity(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		kind     string
		kv       []string
	}{
		"kind only":   {"file", "file", nil},
		"single pair": {`file[path="/etc/foo"]`, "file", []string{"path", "/etc/foo"}},
		"order preserved": {
			`package[name="nginx" manager="apt"]`,
			"package",
			[]string{"name", "nginx", "manager", "apt"},
		},
		"value with spaces":    {`print[message="hello world"]`, "print", []string{"message", "hello world"}},
		"value with quotes":    {`print[message="say \"hi\""]`, "print", []string{"message", `say "hi"`}},
		"value with bracket":   {`print[message="a]b"]`, "print", []string{"message", "a]b"}},
		"empty value":          {`service[name=""]`, "service", []string{"name", ""}},
		"deterministic quoted": {`file[path="/tmp/a\tb"]`, "file", []string{"path", "/tmp/a\tb"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, tensile.AsIdentity(cas.kind, cas.kv...).String())
		})
	}
}

func TestAsIdentity_panics(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		kv []string
	}{
		"odd pairs": {[]string{"path"}},
		"empty key": {[]string{"", "value"}},
		"too many pairs": {[]string{
			"k1", "v", "k2", "v", "k3", "v", "k4", "v",
			"k5", "v", "k6", "v", "k7", "v", "k8", "v",
			"k9", "v",
		}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				tensile.AsIdentity("file", cas.kv...)
			})
		})
	}
}

func TestAsIdentity_comparable(t *testing.T) {
	t.Parallel()

	a := tensile.AsIdentity("file", "path", "/etc/foo")
	b := tensile.AsIdentity("file", "path", "/etc/foo")
	c := tensile.AsIdentity("file", "path", "/etc/bar")

	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)

	m := map[tensile.Identity]bool{a: true}
	assert.True(t, m[b], "equal identity must hit the same map key")
	assert.False(t, m[c])
}

func TestIdentityRegex(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected []string
		input    string
	}{
		"single identity": {
			[]string{`file[path="/etc/foo"]`},
			`level=DEBUG msg=skipping node=file[path="/etc/foo"]`,
		},
		"value with spaces": {
			[]string{`print[message="hello world"]`},
			`executing print[message="hello world"] now`,
		},
		"value with escaped quote": {
			[]string{`print[message="say \"hi\""]`},
			`executing print[message="say \"hi\""] now`,
		},
		"value with bracket": {
			[]string{`print[message="a]b"]`},
			`executing print[message="a]b"] now`,
		},
		"multiple pairs": {
			[]string{`package[name="nginx" manager="apt"]`},
			`node package[name="nginx" manager="apt"] done`,
		},
		"multiple identities": {
			[]string{`file[path="/etc/foo"]`, `dir[path="/etc"]`},
			`file[path="/etc/foo"] depends on dir[path="/etc"]`,
		},
		"no identity": {
			nil,
			`level=INFO msg="starting engine"`,
		},
	}

	re := regexp.MustCompile(tensile.IdentityRegex)

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, re.FindAllString(cas.input, -1))
		})
	}
}

func TestIdentityRegex_matchesAsIdentity(t *testing.T) {
	t.Parallel()

	identity := tensile.AsIdentity("file", "path", `/tmp/a "b" ]c`).String()
	re := regexp.MustCompile(tensile.IdentityRegex)
	assert.Equal(t, identity, re.FindString("node "+identity+" done"))
}

func TestIdentity_LogValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected slog.Value
		identity tensile.Identity
	}{
		"kind only": {
			slog.GroupValue(slog.String("kind", "file")),
			tensile.AsIdentity("file"),
		},
		"pairs": {
			slog.GroupValue(
				slog.String("kind", "package"),
				slog.String("name", "nginx"),
				slog.String("manager", "apt"),
			),
			tensile.AsIdentity("package", "name", "nginx", "manager", "apt"),
		},
		"hostile value": {
			slog.GroupValue(
				slog.String("kind", "file"),
				slog.String("path", `/tmp/a "b" ]c`),
			),
			tensile.AsIdentity("file", "path", `/tmp/a "b" ]c`),
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.True(t, cas.expected.Equal(cas.identity.LogValue()),
				"expected %v, got %v", cas.expected, cas.identity.LogValue())
		})
	}
}

func TestIdentity_LogValue_jsonHandler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("executing", "node", tensile.AsIdentity("file", "path", "/etc/foo"))

	var line struct {
		Node map[string]string `json:"node"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
	assert.Equal(t, map[string]string{"kind": "file", "path": "/etc/foo"}, line.Node)
}
