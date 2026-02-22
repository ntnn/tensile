package tensile

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
)

type Node struct {
	wrapped any
	id      int64
}

func NewNode(input any) (*Node, error) {
	n := new(Node)
	n.wrapped = input

	id, err := hash(input)
	if err != nil {
		return nil, fmt.Errorf("failed to hash input: %w", err)
	}
	n.id = id

	return n, nil
}

func hash(input any) (int64, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal input: %w", err)
	}

	h := fnv.New64a()
	if _, err := h.Write(b); err != nil {
		return 0, fmt.Errorf("failed to write to hash: %w", err)
	}
	return int64(h.Sum64()), nil
}

func (n *Node) ID() int64 {
	return n.id
}
