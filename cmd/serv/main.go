package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/stts-se/segment_checker/dbapi"
	"github.com/stts-se/segment_checker/log"
	"github.com/stts-se/segment_checker/modules"
	"github.com/stts-se/segment_checker/protocol"
)

// Message for sending to client
type Message struct {
	//ClientID    string `json:"client_id"`
	MessageType string `json:"message_type"`
	Payload     string `json:"payload"`
	Error       string `json:"error,omitempty"`
	Info        string `json:"info,omitempty"`
}

func getParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

var chunkExtractor modules.ChunkExtractor

// print serverMsg to server log, and return an http error with clientMsg and the specified error code (http.StatusInternalServerError, etc)
func httpError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
	log.Error(serverMsg)
	http.Error(w, clientMsg, errCode)
}

// print serverMsg to server log, send error message as json to client
func jsonError(w http.ResponseWriter, serverMsg string, clientMsg string, errCode int) {
	log.Error(serverMsg)
	payload := Message{
		Error: clientMsg,
	}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(resJSON))
	return
}

func dbg(format string, args ...interface{}) {
	if *cfg.Debug {
		log.Debug(format, args...)
	}
}

var contextMap = map[string]int64{
	"e":       200,
	"silence": 1000,
}

const fallbackContext = int64(1000)

func load0(sourceFile, userName string, explicitContext int64, w http.ResponseWriter) {
	db.FileMutex.RLock()
	defer db.FileMutex.RUnlock()
	fName := path.Join(*cfg.SourceDataDir, sourceFile)
	bts, err := ioutil.ReadFile(fName)
	if err != nil {
		msg := fmt.Sprintf("Read file failed: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	var segment protocol.SegmentPayload
	err = json.Unmarshal(bts, &segment)
	if err != nil {
		msg := fmt.Sprintf("Unmarshal failed: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	err = db.Lock(segment.UUID, userName)
	if err != nil {
		msg := fmt.Sprintf("Couldn't lock segment: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	var context int64
	if explicitContext > 0 {
		context = explicitContext
	} else if ctx, ok := contextMap[segment.SegmentType]; ok {
		context = ctx
	} else {
		context = fallbackContext
	}

	request := protocol.SplitRequestPayload{
		URL:          segment.URL,
		Chunk:        segment.Chunk,
		SegmentType:  segment.SegmentType,
		LeftContext:  context,
		RightContext: context,
	}
	fmt.Printf("DEBUG %#v\n", request)
	res, err := chunkExtractor.ProcessURLWithContext(request, "")
	if err != nil {
		msg := fmt.Sprintf("chunk extractor failed : %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	res.UUID = segment.UUID
	res.URL = segment.URL
	res.SegmentType = segment.SegmentType

	resJSON, err := json.Marshal(res)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return

	}
	msg := Message{
		MessageType: "audio_chunk",
		Payload:     string(resJSON),
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return

	}
	//log.Info("load output json: %s", string(msgJSON))
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))
}

func load(w http.ResponseWriter, r *http.Request) {
	var err error
	sourceFile := getParam("sourcefile", r)
	userName := getParam("username", r)
	contextS := getParam("context", r)
	if sourceFile == "" {
		msg := fmt.Sprintf("Source file not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	if userName == "" {
		msg := fmt.Sprintf("User name not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	context := int64(-1)
	if contextS != "" {
		context, err = strconv.ParseInt(contextS, 10, 64)
		if err != nil {
			msg := fmt.Sprintf("Couldn't parse int %s", contextS)
			jsonError(w, msg, msg, http.StatusBadRequest)
			return
		}
	}
	log.Info("load | input: %s %s %v", sourceFile, userName, contextS)
	load0(sourceFile, userName, context, w)
}

func next(w http.ResponseWriter, r *http.Request) {
	var err error
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	log.Info("next | input: %s", string(body))

	query := protocol.QueryPayload{}
	err = json.Unmarshal(body, &query)
	if err != nil {
		msg := fmt.Sprintf("failed to unmarshal incoming JSON : %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}

	if query.UserName == "" {
		msg := fmt.Sprintf("User name not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	if query.StepSize == 0 {
		msg := fmt.Sprintf("Step size not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	if query.CurrID == "undefined" {
		query.CurrID = ""
	}
	sourceFile, err := db.GetNextCheckableSegment(query)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	load0(sourceFile, query.UserName, query.Context, w)
}

func release(w http.ResponseWriter, r *http.Request) {
	uuid := getParam("uuid", r)
	userName := getParam("username", r)
	if uuid == "" {
		msg := fmt.Sprintf("uuid not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	if userName == "" {
		msg := fmt.Sprintf("User name not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	log.Info("release | input: %s %s", uuid, userName)

	err := db.Unlock(uuid, userName)
	if err != nil {
		msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	msg := Message{Info: fmt.Sprintf("Unlocked segment %s for user %s", uuid, userName)}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))
}

func releaseAll(w http.ResponseWriter, r *http.Request) {
	userName := getParam("username", r)
	if userName == "" {
		msg := fmt.Sprintf("User name not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	log.Info("load | input: %s", userName)

	n, err := db.UnlockAll(userName)
	if err != nil {
		msg := fmt.Sprintf("failed to unlock : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}

	msg := Message{Info: fmt.Sprintf("Unlocked %d segments for user %s", n, userName)}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))
}

func save(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	log.Info("save | input: %s", string(body))

	annotation := protocol.AnnotationPayload{}
	err = json.Unmarshal(body, &annotation)
	if err != nil {
		msg := fmt.Sprintf("failed to unmarshal incoming JSON : %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}

	err = db.Save(annotation)
	if err != nil {
		msg := fmt.Sprintf("failed to save annotation : %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}

	msg := Message{Info: fmt.Sprintf("Saved annotation for segment with id %s", annotation.UUID)}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
		return
	}
	// log.Info("save output json: %s", string(msgJSON))
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))
}

func stats(w http.ResponseWriter, r *http.Request) {
	res, err := db.Stats()
	if err != nil {
		msg := fmt.Sprintf("failed to get stats from db : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	msg := Message{
		MessageType: "stats",
		Payload:     string(resJSON),
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
		return

	}
	//log.Info("load output json: %s", string(msgJSON))
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))

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

var cfg = &Config{}
var db *dbapi.DBAPI

// Config for server
type Config struct {
	Host              *string `json:"host"`
	Port              *string `json:"port"`
	ServeDir          *string `json:"static_dir"`
	SourceDataDir     *string `json:"source_data_dir"`
	AnnotationDataDir *string `json:"annotation_data_dir"`
	Debug             *bool   `json:"debug"`
}

func main() {
	var err error

	cmd := path.Base(os.Args[0])

	// Flags
	cfg.Host = flag.String("h", "localhost", "Server `host`")
	cfg.Port = flag.String("p", "7371", "Server `port`")
	cfg.ServeDir = flag.String("serve", "static", "Serve static `folder`")
	cfg.SourceDataDir = flag.String("source", "", "Source data `folder`")
	cfg.AnnotationDataDir = flag.String("annotation", "", "Annotation data `folder`")

	cfg.Debug = flag.Bool("debug", false, "Debug mode")
	protocol := "http"

	help := flag.Bool("help", false, "Print usage and exit")
	flag.Parse()

	cfgJSON, _ := json.Marshal(cfg)
	log.Info("Server config: %#v", string(cfgJSON))

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

	if *cfg.SourceDataDir == "" {
		fmt.Fprintf(os.Stderr, "Required flag source not set\n")
		flag.Usage()
		os.Exit(1)
	}
	if *cfg.AnnotationDataDir == "" {
		fmt.Fprintf(os.Stderr, "Required flag annotation not set\n")
		flag.Usage()
		os.Exit(1)
	}

	db = dbapi.NewDBAPI(*cfg.SourceDataDir, *cfg.AnnotationDataDir)
	err = db.LoadData()
	if err != nil {
		log.Fatal("Couldn't load data: %v", err)
	}

	chunkExtractor, err = modules.NewChunkExtractor()
	if err != nil {
		log.Fatal("Couldn't initialize chunk extractor: %v", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/save/", save).Methods("POST")
	//r.HandleFunc("/load/{sourcefile}/{username}", load).Methods("GET")
	//r.HandleFunc("/next/{currid}/{username}", next).Methods("GET")
	r.HandleFunc("/load/{sourcefile}", load).Methods("GET")
	r.HandleFunc("/next/", next).Methods("POST")
	r.HandleFunc("/release/{uuid}/{username}", release).Methods("GET")
	r.HandleFunc("/releaseall/{username}", releaseAll).Methods("GET")
	r.HandleFunc("/stats/", stats).Methods("GET")
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
		log.Error(msg)
		return
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(*cfg.ServeDir))))

	srv := &http.Server{
		Handler:      r,
		Addr:         *cfg.Host + ":" + *cfg.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Info("Server started on %s://%s", protocol, *cfg.Host+":"+*cfg.Port)
	log.Info("Serving folder %s", *cfg.ServeDir)
	if err = srv.ListenAndServe(); err != nil {
		log.Fatal("Server failure: %v", err)
	}
}
