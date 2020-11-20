package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/mux"

	"github.com/stts-se/segment_checker/modules"
	"github.com/stts-se/segment_checker/protocol"
)

var chunkExtractor modules.ChunkExtractor

// print serverMsg to server log, and return an http error with clientMsg and the specified error code (http.StatusInternalServerError, etc)
func httpError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
	log.Printf("error : %s", serverMsg)
	http.Error(w, clientMsg, errCode)
}

func echoJSON(w http.ResponseWriter, r *http.Request) {
	log.Println("echoJSON", r.Method)
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Printf("echoJSON input json: %s", string(body))
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(body))
}

func extractChunk(w http.ResponseWriter, r *http.Request) {
	log.Println("extractChunk", r.Method)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Printf("extractChunk input json: %s", string(body))

	source := protocol.SplitRequestPayload{}
	err = json.Unmarshal(body, &source)
	if err != nil {
		msg := fmt.Sprintf("failed to unmarshal incoming JSON : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
		return
	}
	res, err := chunkExtractor.ProcessURLWithContext(source, "")
	if err != nil {
		msg := fmt.Sprintf("chunk extractor failed : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
		return
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(resJSON))
}

var walkedURLs []string

func generateDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<html><head><title>%s</title></head><body>", "STTS PromptRec: Doc")
	for _, url := range walkedURLs {
		fmt.Fprintf(w, "%s<br/>\n", url)
	}
	fmt.Fprintf(w, "</body></html>")
}

func main() {
	var err error

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

	chunkExtractor, err = modules.NewChunkExtractor()
	if err != nil {
		log.Fatalf("Couldn't initialize chunk extractor: %v", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/extract_chunk/", extractChunk).Methods("POST", "OPTIONS")
	r.HandleFunc("/echo_json/", echoJSON).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/doc/", generateDoc).Methods("GET")

	docs := make(map[string]string)
	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		t, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		if info, ok := docs[t]; ok {
			t = fmt.Sprintf("%s - %s", t, info)
		}
		walkedURLs = append(walkedURLs, t)
		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("failure to walk URLs : %v", err)
		log.Println(msg)
		return
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(*serveDir))))

	srv := &http.Server{
		Handler:      r,
		Addr:         *host + ":" + *port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Server started on %s://%s", protocol, *host+":"+*port)
	log.Printf("Serving folder %s", *serveDir)
	log.Fatal(srv.ListenAndServe())
}
