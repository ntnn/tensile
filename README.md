# tensile

tensile is a yak shaving exercise to write a config management ecosystem
a la Puppet, Ansible, Chef etc.pp. in Go.

The existing config management solutions all have hefty pros and cons
and require either infrastructure, hacks, wrappers and/or get very
complicated the more complex the deployment becomes.

tensile isn't supposed to be a one-size-fits-all solution but rather
a library with which it becomes easy to implement requirements - all
with the added benefit of Go tooling.

## Design

1. Shape
1. Node
1. Queue
1. Engine

### Shape

A shape is an abstract name of a type of Node, e.g. Service for
sysV/systemd/etc.pp. services, File for files or directories, etc.pp.

These are used together with a Nodes name to identify collisions.

### Node

A Node is an element to manage a resource like a file or to execute a command, hence similar to resources in Puppet or modules in Ansible.

```go
// A file should exist at path /an/ex/ample with the content "Hello,
// world!"
myFile := &nodes.File{
	Target: "/an/ex/ample",
	Content: "Hello, world!",
}
// A directory at path /an should exist.
myDir := &nodes.Dir{
	Target: "/an/ex",
}
```

### Queue

The Queue is the only non-interchangeable part and used to queue and
order nodes for execution.

### Engine

Engines manage the execution and state of nodes.

Engine can be implemented differently for e.g. parallelisation or for
used in clusters.

```go
// The simple engine sequentally realizes all elements.
simple := engines.NewSimple()

// The nodes from the previous example are added.
// The order does not matter - the queue figures out the order as
// needed.
// If the validation of any of the passed elements fails those errors
// will be returned.
if err := simple.Queue.Add(myFile, myDir); err != nil {
        log.Fatal(err)
}

// All that's left is letting the engine run.
if err := simple.Run(context.Background()); err != nil {
        log.Fatal(err)
}
```

As engines work with interfaces nodes can be anything that satisfies the
relevant interfaces - and since elements are written in Go no other tool
is needed to manage dependencies.

## Compliance

Golang is ideally suited to ensure compliance of e.g. CIS or internal
standards of all kinds of systems.

A single library can be written that maintains the status quo of the
compliance requirements and either included in the binaries deploying
e.g. applications or in a binary that does nothing but ensure
compliance.

```go
simple := engines.NewSimple()

// Assuming a module or package tensilecis where Nodes() returns
// a slice of Nodes.
if err := simple.Add(tensilecis.Nodes()...); err != nil {
    log.Fatal(err)
}

// Add other nodes to deploy applications, agents, etc.pp.
if err := simple.Add(otherNodes...); err != nil {
    log.Fatal(err)
}

if err := simple.Run(context.Background()); err != nil {
    log.Fatal(err)
}
```

These binaries can be built once for the target CPU architecture and
run on physical or virtual machines, in the lifecycle of OS images and
containers - even on unixoid appliances.

No need to set up AWX or a Puppet Master, no need for dependency
management or a separate host to run from.

## Cluster-awareness

A flexible approach to engines allows to deploy clustered applications
easily.

Cluster members in a tensile cluster-aware engine could authenticate
with each other through a pre shared key and exchange certificates for
communication - e.g. to exchange secrets for the applications and
services to deploy - and even manage the execution of the entire cluster
to ensure that e.g. one node is deployed first before other nodes are
set up and connected to the initial node.

Examples for this could be glusterfs, kubernetes, vault etc.pp. - any
clustered applications that requires synchronization and key exchange
during setup.
