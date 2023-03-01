# gorrect your systems

gorrect is a yak shaving exercise to write a config management ecosystem
a la Puppet, Ansible, Chef etc.pp. in go.

The existing config management solutions all have hefty pros and cons
and require either infrastructure, hacks, wrappers and/or get very
complicated the more complex the deployment becomes.

gorrect isn't supposed to be a one-size-fits-all solution but rather
a library with which it becomes easy to implement requirements - all
with the added benefit of Go tooling.

## Design

gorrect has two major pieces: Elements and Engines.

Elements are similar to resources in Puppet or modules in Ansible.

```go
// A file should exist at path /an/ex/ample with the content "Hello,
// world!"
myFile := &gorrect.File{
	Target: "/an/ex/ample",
	Content: "Hello, world!",
}
// A directory at path /an should exist.
myDir := &gorrect.Dir{
	Target: "/an/ex",
}
```

Engines are the executing part that decides in which order elements are
realized and realizes them.

```go
// The simple engine sequentally realizes all elements.
simple := engines.NewSimple()

// The elements from the previous example are added.
// The order does not matter - the engine figures out the order as
// needed.
// If the validation of any of the passed elements fails those errors
// will be returned.
if err := simple.Add(myFile, myDir); err != nil {
        log.Fatal(err)
}

// All that's left is letting the engine run.
if err := simple.Run(context.Background()); err != nil {
        log.Fatal(err)
}
```

As engines work with interfaces elements can be anything that satisfies
the relevant interfaces - and since elements are written in Go no other
tool is needed to manage dependencies.

## Compliance

Golang is ideally suited to ensure compliance of e.g. CIS or internal
standards of all kinds of systems.

A single library can be written that maintains the status quo of the
compliance requirements and either included in the binaries deploying
e.g. applications or in a binary that does nothing but ensure
compliance.

```go
simple := engines.NewSimple()

// Assuming a module or package gorrectcis where Elements() returns
// a channel of elements.
if err := simple.AddFrom(gorrectcis.Elements()); err != nil {
    log.Fatal(err)
}

// Add other elements to deploy applications, agents, etc.pp.
if err := simple.Add(otherElements...); err != nil {
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

Cluster members in a gorrect cluster-aware engine could authenticate
with each other through a pre shared key and exchange certificates for
communication - e.g. to exchange secrets for the applications and
services to deploy - and even manage the execution of the entire cluster
to ensure that e.g. one node is deployed first before other nodes are
set up and connected to the initial node.

Examples for this could be glusterfs, kubernetes, vault etc.pp. - any
clustered applications that requires synchronization and key exchange
during setup.
