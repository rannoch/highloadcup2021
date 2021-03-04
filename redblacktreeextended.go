package main

import rbt "github.com/emirpasic/gods/trees/redblacktree"

// RedBlackTreeExtended to demonstrate how to extend a RedBlackTree to include new functions
type RedBlackTreeExtended struct {
	maxNodeValue interface{}
	*rbt.Tree
}

func NewRedBlackTreeExtended(tree *rbt.Tree) *RedBlackTreeExtended {
	return &RedBlackTreeExtended{Tree: tree}
}

func (tree *RedBlackTreeExtended) Put(key interface{}, value interface{}) {
	tree.Tree.Put(key, value)

	compare := tree.Comparator(key, tree.maxNodeValue)
	if compare > 0 {
		tree.maxNodeValue = value
	}
}

func (tree *RedBlackTreeExtended) GetMaxNodeValue() *ReportTree {
	if tree.maxNodeValue == nil {
		return nil
	}

	return tree.maxNodeValue.(*ReportTree)
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

		if node.Left != nil {
			tree.maxNodeValue = node.Left.Value
		} else if node.Parent != nil {
			tree.maxNodeValue = node.Parent.Value
		} else {
			tree.maxNodeValue = nil
		}
		return node.Value
	}
	return nil
}
