// Command wartable is a CLI for playing Wartable by order files
// (docs/dev-plan.md section 7, M5). It must be run from the repository
// root: it reads rules data from data/ with paths relative to the
// working directory.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = runNew(os.Args[2:])
	case "turn":
		err = runTurn(os.Args[2:])
	case "show":
		err = runShow(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wartable:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  wartable new <state.json>")
	fmt.Fprintln(os.Stderr, "  wartable turn <state.json> <north-orders.txt> <south-orders.txt>")
	fmt.Fprintln(os.Stderr, "  wartable show <state.json>")
}
