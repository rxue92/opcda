package opcda_test

import (
	"testing"

	"github.com/rxue92/opcda"
)

func testingCreateNewTree() *opcda.Tree {

	root := opcda.Tree{
		Name:     "root",
		Parent:   nil,
		Branches: []*opcda.Tree{},
		Leaves: []opcda.Leaf{
			{
				Name:   "bandwidth",
				ItemId: "bandwidth",
			},
		},
	}
	options := opcda.Tree{
		Name:     "options",
		Parent:   &root,
		Branches: []*opcda.Tree{},
		Leaves: []opcda.Leaf{
			{
				Name:   "frequency",
				ItemId: "options.frequency",
			},
			{
				Name:   "amplitute",
				ItemId: "options.amplitude",
			},
		},
	}
	numeric := opcda.Tree{
		Name:     "numeric",
		Parent:   &root,
		Branches: []*opcda.Tree{},
		Leaves: []opcda.Leaf{
			{
				Name:   "sin",
				ItemId: "numeric.sin",
			},
			{
				Name:   "cos",
				ItemId: "numeric.cos",
			},
			{
				Name:   "tan",
				ItemId: "numeric.tan",
			},
		},
	}
	sim := opcda.Tree{
		Name:   "sim",
		Parent: &root,
	}
	dev1 := opcda.Tree{
		Name:   "dev1",
		Parent: &sim,
		Leaves: []opcda.Leaf{
			{
				Name:   "t1",
				ItemId: "sim.dev1.t1",
			},
			{
				Name:   "t2",
				ItemId: "sim.dev1.t2",
			},
		},
	}

	sim.Branches = append(sim.Branches, &dev1)
	root.Branches = append(root.Branches, &options, &numeric, &sim)
	return &root
}

func TestTreeExtractBranchByName(t *testing.T) {
	tree := testingCreateNewTree()
	subtree := opcda.ExtractBranchByName(tree, "numeric")
	if subtree == nil {
		t.Fatal("subtree not correctly extracted")
	}
	if len(opcda.CollectTags(subtree)) != 3 {
		t.Fatal("subtree not correctly extracted")
	}
}

func TestTreeCollectTags(t *testing.T) {
	tree := testingCreateNewTree()
	collection := opcda.CollectTags(tree)
	if len(collection) != 8 {
		t.Fatal("not enough tags collected")
	}
}

func TestExtractBranchByNames(t *testing.T) {
	tree := testingCreateNewTree()
	subTree := opcda.ExtractBranchByNames(tree, "sim", "dev1")
	if subTree == nil {
		t.Fatal("subtree not correctly extracted")
	}
	if len(opcda.CollectTags(subTree)) != 2 {
		t.Fatal("subtree not correctly extracted")
	}
}
