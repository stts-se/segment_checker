package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
)

// print serverMsg to server log, and return an http error with clientMsg and the specified error code (http.StatusInternalServerError, etc)
func httpError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
	log.Printf("error : %s", serverMsg)
	http.Error(w, clientMsg, errCode)
}

func main() {

	cmd := path.Base(os.Args[0])

	// Flags
	host := flag.String("h", "localhost", "Server `host`")
	port := flag.String("p", "7371", "Server `port`")
	serveDir := flag.String("s", "static", "Serve `folder`")
	protocol := "http"
	help := flag.Bool("help", false, "Print usage and exit")
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <options> <folder to serve>\n", cmd)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if len(flag.Args()) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	http.Handle("/", http.FileServer(http.Dir(*serveDir)))
	log.Printf("Server started on %s://%s", protocol, *host+":"+*port)
	log.Printf("Serving folder %s", *serveDir)
	http.ListenAndServe(*host+":"+*port, nil)
}
