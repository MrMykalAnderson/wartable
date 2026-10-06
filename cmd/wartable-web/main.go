// Command wartable-web is a local web viewer for Wartable games
// (docs/dev-plan.md section 7, M6): it draws the hex map with units and
// facing, loads a saved game, and steps through a turn's events one at a
// time. It must be run from the repository root, same as cmd/wartable:
// it reads rules data from data/ and serves the frontend from web/, both
// relative to the working directory.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/game", handleGame)
	mux.HandleFunc("/api/turn", handleTurn)
	mux.HandleFunc("/api/options", handleOptions)
	mux.HandleFunc("/api/predict", handlePredict)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	fmt.Printf("wartable-web listening on http://%s (serving web/ from the current directory)\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
