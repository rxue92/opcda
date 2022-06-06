package opcda

import (
	"testing"
)

func testingCreateNewTree() *Tree {

	root := Tree{
		Name:     "root",
		Parent:   nil,
		Branches: []*Tree{},
		Leaves: []Leaf{
			{
				Name:   "bandwidth",
				ItemId: "bandwidth",
			},
		},
	}
	options := Tree{
		Name:     "options",
		Parent:   &root,
		Branches: []*Tree{},
		Leaves: []Leaf{
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
	numeric := Tree{
		Name:     "numeric",
		Parent:   &root,
		Branches: []*Tree{},
		Leaves: []Leaf{
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

	root.Branches = append(root.Branches, &options, &numeric)
	return &root
}

func TestTreeExtractBranchByName(t *testing.T) {
	tree := testingCreateNewTree()
	subtree := ExtractBranchByName(tree, "numeric")
	if subtree == nil {
		t.Fatal("subtree not correctly extracted")
	}
	if len(CollectTags(subtree)) != 3 {
		t.Fatal("subtree not correctly extracted")
	}
}

func TestTreeCollectTags(t *testing.T) {
	tree := testingCreateNewTree()
	collection := CollectTags(tree)
	if len(collection) != 6 {
		t.Fatal("not enough tags collected")
	}
}
