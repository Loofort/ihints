package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

type IndexTree struct {
	name   []byte
	depth  int
	offset int64
	childs [256]*IndexTree
}

func (it *IndexTree) IsLeaf() bool {
	for _, node := range it.childs {
		if node != nil {
			return false
		}
	}
	return true
}

func NewIndexFromFile(hintsFile string) (*IndexTree, error) {
	root := &IndexTree{}

	f, err := os.OpenFile(hintsFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var query []byte
	offset := int64(0)
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		offset += int64(len(line))

		pices := bytes.SplitN(line, []byte{'\t'}, 3)
		if len(pices) != 3 {
			return nil, fmt.Errorf("incorrect line: %s", line)
		}
		if bytes.Compare(query, pices[1]) == 0 {
			continue
		}

		// find new query
		query = pices[1]
		ofs := offset - int64(len(line))
		if AddNode(root, query, ofs) == nil {
			return nil, fmt.Errorf("unable to add node %s offset %d", query, ofs)
		}
	}

	return root, nil
}

func FindNode(it *IndexTree, q []byte) (*IndexTree, bool) {
	for _, char := range q {
		child := it.childs[char]
		if child == nil {
			return it, false
		}

		it = child
	}
	return it, true
}

func AddNode(it *IndexTree, q []byte, offset int64) *IndexTree {
	node, ok := FindNode(it, q)
	if ok {
		// node already exists
		return nil
	}

	d := node.depth
	for i, char := range q[d:] {
		child := &IndexTree{
			name:  q[:d+i+1],
			depth: d + i + 1,
		}

		node.childs[char] = child
		node = child
	}

	node.offset = offset
	return node
}

func WalkTree(node *IndexTree, foo func(*IndexTree)) {
	nodes := node.childs[:]
	for len(nodes) > 0 {
		node, nodes = nodes[0], nodes[1:]
		if node != nil {
			foo(node)
			nodes = append(nodes, node.childs[:]...)
		}
	}
}
