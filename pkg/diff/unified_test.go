package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnified_String(t *testing.T) {
	t.Parallel()

	u := Unified{
		Path: "/etc/motd",
		Old:  "hello\n",
		New:  "world\n",
	}
	assert.Equal(t, `--- /etc/motd
+++ /etc/motd
@@ -1 +1 @@
-hello
+world
`, u.String())
}
