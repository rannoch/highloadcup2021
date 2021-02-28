package main

import rbt "github.com/emirpasic/gods/trees/redblacktree"

// RedBlackTreeExtended to demonstrate how to extend a RedBlackTree to include new functions
type RedBlackTreeExtended struct {
	*rbt.Tree
}

// GetMax gets the max value and flag if found
func (tree *RedBlackTreeExtended) GetMax() interface{} {
	node, _ := tree.getMaxFromNode(tree.Root)
	if node != nil {
		return node.Value
	}
	return nil
}

func (tree *RedBlackTreeExtended) getMaxFromNode(node *rbt.Node) (foundNode *rbt.Node, found bool) {
	if node == nil {
		return nil, false
	}
	if node.Right == nil {
		return node, true
	}
	return tree.getMaxFromNode(node.Right)
}

// RemoveMax removes the max value and flag if found
func (tree *RedBlackTreeExtended) RemoveMax() interface{} {
	node, found := tree.getMaxFromNode(tree.Root)
	if found {
		tree.Remove(node.Key)
		return node.Value
	}
	return nil
}
