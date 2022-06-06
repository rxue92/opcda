package main

import (
	"fmt"

	"github.com/rxue92/opcda"
)

func main() {
	progid := "Graybox.Simulator"
	nodes := []string{"localhost"}

	// create browser and collect all tags on OPC server
	browser, err := opcda.CreateBrowser(progid, nodes)
	if err != nil {
		panic(err)
	}

	// extract subtree
	subtree := opcda.ExtractBranchByName(browser, "textual")

	// print out all the information
	opcda.PrettyPrint(subtree)

	// create opc connection with all tags from subtree
	conn, _ := opcda.NewConnection(
		progid,
		nodes,
		opcda.CollectTags(subtree),
	)
	defer conn.Close()

	fmt.Println(conn.Read())
}
