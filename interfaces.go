package tensile

// Identifier is the interface that must be implemented by all [Node].
type Identifier interface {
	// Identity returns the node's identity.
	// It must be deterministic and build from [AsIdentity].
	Identity() Identity
}

// Validator is the interface to be satisfied by a [Node] when the
// configuration needs to be validated e.g. before execution.
type Validator interface {
	// Validate validates the configuration of the node.
	// It may be used to setup states in the node.
	Validate(wire Wire) error
}

// Conflictor is the interface to be satisfied by a [Node] when it touches resources beyond its own identity.
//
// e.g. std.Symlink and std.FileContent both touch the file at their path.
//
// A node implicitly conflicts with its own identity.
// Two nodes sharing any conflict identity is an error when building the final graph.
type Conflictor interface {
	// Conflicts returns the identities of resources the node touches,
	// e.g. packages or files.
	Conflicts() ([]Identity, error)
}

// Depender is the interface to be satisfied by a [Node] when it depends
// on other resources to be provided by other [Node]., e.g. a package
// that must be installed or a file to be ensured.
type Depender interface {
	// DependsOn returns a list of resources the node depends on, e.g.
	// packages or files.
	DependsOn() ([]Identity, error)
}

// Notifier is the interface to be satisfied by a [Node] when it
// notifies handlers, e.g. restarting a service after changing its
// configuration.
type Notifier interface {
	// Notifies returns a list of resources provided by handlers to
	// notify.
	Notifies() ([]Identity, error)
}

// Serializer is the interface to be satisfied by a [Node] whose
// lifecycle must not overlap with other nodes with the same key.
//
// This is resolved during scheduling in the work queue, not at runtime in the engine.
// Meaning for each key yielded by a number of nodes even when multiple
// of them are ready only one of them is being scheduled to be executed
// at a time.
//
// E.g. package managers often have an internal lock that prevents
// parallel execution of Package nodes.
type Serializer interface {
	// SerializesOn returns the key to serialize on.
	// Empty means no serialization.
	SerializesOn() string
}

// Reporter is the interface to be satisfied by a [Node] when it reports an output.
type Reporter interface {
	// Report returns the node's output.
	// It is called after the node completed and must not modify the system.
	Report(wire Wire) (any, error)
}

// Executor is the interface to be satisfied by a [Node] to be executed.
type Executor interface {
	// NeedsExecution is run before Execute. NeedsExecution must not
	// mane any modifications to the system, it must be a stateless
	// check to check if Node.Execute must be called.
	//
	// NeedsExecution is called e.g. for noop runs to check if any
	// changes are needed.
	NeedsExecution(wire Wire) (bool, error)
	// Execute is called for the node to make the desired change.
	Execute(wire Wire) error
}
