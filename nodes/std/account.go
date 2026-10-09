package std

import (
	"fmt"

	"github.com/ntnn/tensile/pkg/shadow"
)

// stateField is the diff field of account presence.
const stateField = "state"

// accountSerializeKey serializes nodes changing local accounts, the tools lock passwd and group.
const accountSerializeKey = "account"

// accountService returns svc, or the [shadow.Service] for the available tools when svc is nil.
func accountService(svc shadow.Service) (shadow.Service, error) {
	if svc != nil {
		return svc, nil
	}
	svc, err := shadow.New(shadow.Options{})
	if err != nil {
		return nil, fmt.Errorf("setting up account tools: %w", err)
	}
	return svc, nil
}
