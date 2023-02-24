package engines

import "fmt"

var (
	ErrSameIdentityAlreadyRegistered = fmt.Errorf("an element with the same identity is already registered")
)
