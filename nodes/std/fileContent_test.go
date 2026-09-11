package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileContent_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		onDisk   *string
		content  string
	}{
		"missing file":       {true, nil, "content"},
		"matching content":   {false, new("content"), "content"},
		"differing content":  {true, new("other"), "content"},
		"empty vs non-empty": {true, new(""), "content"},
		"both empty":         {false, new(""), ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "file")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(path, []byte(*cas.onDisk), 0o600))
			}

			f := &FileContent{
				Path:    path,
				Content: cas.content,
			}

			needs, err := f.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}
