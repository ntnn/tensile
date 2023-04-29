package tensile

import (
	"fmt"
)

func FormatIdentity(shape Shape, identifier string) string {
	return fmt.Sprintf("%s[%s]", shape, identifier)
}
