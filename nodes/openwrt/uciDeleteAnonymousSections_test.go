package openwrt

import (
	"context"
	"errors"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWire(t *testing.T) tensile.Wire {
	t.Helper()
	return &tensile.DefaultWire{Ctx: t.Context()}
}

const showDhcp = `dhcp.@dnsmasq[0]=dnsmasq
dhcp.@dnsmasq[0].domainneeded='1'
dhcp.named=host
dhcp.named.name='named'
dhcp.@host[1]=host
dhcp.@host[1].name='first'
dhcp.@host[2]=host
dhcp.@host[2].name='second'
`

// fakeUCIShow answers uci show for a config and its sections.
// Section ids are mapped from extended paths, deletions recorded.
func fakeUCIShow(t *testing.T, deleted *[]string) func(context.Context, ...string) ([]byte, error) {
	t.Helper()
	sections := map[string]string{
		"dhcp.@dnsmasq[0]": "dhcp.cfg01411c=dnsmasq",
		"dhcp.@host[1]":    "dhcp.cfg02host=host",
		"dhcp.@host[2]":    "dhcp.cfg03host=host",
	}
	return func(_ context.Context, args ...string) ([]byte, error) {
		switch args[0] {
		case "show":
			if args[1] == "dhcp" {
				return []byte(showDhcp), nil
			}
			line, ok := sections[args[1]]
			require.True(t, ok, "unexpected uci show %q", args[1])
			return []byte(line + "\n"), nil
		case "delete":
			*deleted = append(*deleted, args[1])
			return nil, nil
		default:
			t.Fatalf("unexpected uci verb %q", args[0])
			return nil, nil
		}
	}
}

func TestUCIDeleteAnonymousSections_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&UCIDeleteAnonymousSections{Config: "c", Type: "t"}).Validate(nil))
	require.Error(t, (&UCIDeleteAnonymousSections{Type: "t"}).Validate(nil))
	require.Error(t, (&UCIDeleteAnonymousSections{Config: "c"}).Validate(nil))
}

func TestUCIDeleteAnonymousSections_NeedsExecution(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		typ     string
		out     string
		err     error
	}{
		"anonymous sections exist": {
			want: true,
			typ:  "host",
			out:  showDhcp,
		},
		"only named sections": {
			typ: "host",
			out: "dhcp.named=host\ndhcp.named.name='named'\n",
		},
		"no sections of type": {
			typ: "domain",
			out: showDhcp,
		},
		"missing config": {
			typ: "host",
			out: "uci: Entry not found",
			err: errRun,
		},
		"run failure": {
			wantErr: true,
			typ:     "host",
			out:     "boom",
			err:     errRun,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			del := &UCIDeleteAnonymousSections{
				Config: "dhcp",
				Type:   cas.typ,
				run: func(_ context.Context, _ ...string) ([]byte, error) {
					return []byte(cas.out), cas.err
				},
			}

			got, _, err := del.NeedsExecution(testWire(t))
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestUCIDeleteAnonymousSections_Execute(t *testing.T) {
	t.Parallel()

	var deleted []string
	del := &UCIDeleteAnonymousSections{
		Config: "dhcp",
		Type:   "host",
		run:    fakeUCIShow(t, &deleted),
	}

	_, err := del.Execute(testWire(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"dhcp.cfg02host", "dhcp.cfg03host"}, deleted,
		"anonymous sections deleted by id, named section kept")
}
