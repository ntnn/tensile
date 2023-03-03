package gorrect

import "fmt"

// Identity is - loosely defined - the type of resource modified by an
// element. Together with the string returned by an Identitier it
// defines the full identity of an element.
//
// This is meant to prevent clashes when multiple elements modify the
// same resource.
//
// E.g. consider two elements:
// templated := &gorrect.Template{
//   Path: "/path/to/file",
//   ...
// }
//
// lineInFile := &gorrect.LineInFile{
//   Path: "/path/to/target",
// }
//
// Both would have the identity path[/path/to/target] and would create
// an error when adding to an engine.
//
// If an element may have the same identity as another element it can
// implement the NoIdentityClasher interface.
type Identity string

const (
	Noop    Identity = "noop"
	Path    Identity = "path"
	Package Identity = "package"
	Service Identity = "service"
)

func FormatIdentitier(ident Identitier) string {
	i, s := ident.Identity()
	return fmt.Sprintf("%s[%s]", i, s)
}
