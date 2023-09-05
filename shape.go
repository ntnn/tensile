package tensile

// Shape is - loosely defined - the type of resource modified by an
// element. Together with the string returned by a Identitier it defines
// the identity of an element.
//
// This is meant to prevent clashes when multiple elements modify the
// same resource.
//
// E.g. consider two elements:
//
//	templated := &tensile.Template{
//	  Path: "/path/to/file",
//	  ...
//	}
//
//	lineInFile := &tensile.LineInFile{
//	  Path: "/path/to/target",
//	}
//
// Both would have the identity path[/path/to/target] and would create
// an error when adding to a queue.
type Shape string

const (
	Noop    Shape = "noop"
	Path    Shape = "path"
	Package Shape = "package"
	Service Shape = "service"
)
