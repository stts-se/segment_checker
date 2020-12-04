package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

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

type AnnotationReleaseAndQueryPayload struct {
	Annotation protocol.AnnotationPayload `json:"annotation"`
	Release    protocol.ReleasePayload    `json:"release"`
	Query      protocol.QueryPayload      `json:"query"`
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

// print serverMsg to server log, send client message over websocket
func wsError(conn *websocket.Conn, serverMsg string, clientMsg string) {
	log.Error(serverMsg)
	payload := Message{
		Error: clientMsg,
	}
	resJSON, err := json.Marshal(payload)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		log.Error(msg)
		return
	}
	conn.WriteMessage(websocket.TextMessage, resJSON)
	return
}

// print serverMsg to server log, send error message as json to client
func jsonError(w http.ResponseWriter, serverMsg string, clientMsg string) {
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

var clientMutex sync.RWMutex
var clients = make(map[string]*websocket.Conn)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["client_id"]
	if clientID == "" {
		msg := "expected client ID, got empty string"
		httpError(w, msg, msg, http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("failed to upgrade HTTP request to websocket : %v", err)
	}

	clientMutex.Lock()
	clients[clientID] = ws
	clientMutex.Unlock()
	log.Info("added client websocket for %s", clientID)

	// listen forever
	go listenToClient(ws)
}

func wsPayload(conn *websocket.Conn, msgType string, payload interface{}) {
	bts, err := json.Marshal(payload)
	if err != nil {
		log.Error("failed to marshal struct into JSON : %v", err)
		return
	}
	resp := Message{
		//ClientID:    msg.ClientID,
		MessageType: msgType,
		Payload:     string(bts),
	}

	jsnMsg, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}
	conn.WriteMessage(websocket.TextMessage, jsnMsg)
}

func wsInfo(conn *websocket.Conn, msg string) {
	resp := Message{
		//ClientID:    msg.ClientID,
		Info: msg,
	}

	jsnMsg, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}
	conn.WriteMessage(websocket.TextMessage, jsnMsg)
}

func listenToClient(conn *websocket.Conn) {
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			msg := fmt.Sprintf("websocket error : %v", err)
			fmt.Println("???", msg)
			wsError(conn, msg, msg)
			return
		}

		log.Info("Got %#v\n", msg)

		switch msg.MessageType {
		case "stats":
			res, err := db.Stats()
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}
			wsPayload(conn, "stats", res)

		case "next":
			var payload protocol.QueryPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}

			if payload.UserName == "" {
				msg := fmt.Sprintf("User name not provided for query")
				wsError(conn, msg, msg)
				return
			}
			if payload.StepSize == 0 {
				msg := fmt.Sprintf("Step size not provided for query")
				wsError(conn, msg, msg)
				return
			}
			if payload.CurrID == "undefined" {
				payload.CurrID = ""
			}
			sourceFile, err := db.GetNextSegment(payload, true)
			if err != nil {
				msg := fmt.Sprintf("%v", err)
				wsError(conn, msg, msg)
				return
			}
			load(conn, sourceFile, payload.UserName, payload.Context)

		case "savereleaseandnext":
			var payload AnnotationReleaseAndQueryPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}
			saveReleaseAndNext(conn, payload)

		case "release":
			var payload protocol.ReleasePayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}

			err = db.Unlock(payload.UUID, payload.UserName)
			if err != nil {
				msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
				wsError(conn, msg, msg)
				return
			}
			msg := fmt.Sprintf("Unlocked segment %s for user %s", payload.UUID, payload.UserName)
			wsInfo(conn, msg)

		case "release_all":
			var payload protocol.ReleasePayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}

			n, err := db.UnlockAll(payload.UserName)
			if err != nil {
				msg := fmt.Sprintf("Failed to unlock : %v", err)
				wsError(conn, msg, msg)
				return
			}
			msg := fmt.Sprintf("Unlocked %d segments for user %s", n, payload.UserName)
			wsInfo(conn, msg)

		default:
			log.Error("Unknown message type: %s", msg.MessageType)
		}
	}
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

func load0(annotation protocol.AnnotationPayload, userName string, explicitContext int64, w http.ResponseWriter) {
	var context int64
	if explicitContext > 0 {
		context = explicitContext
	} else if ctx, ok := contextMap[annotation.SegmentType]; ok {
		context = ctx
	} else {
		context = fallbackContext
	}

	request := protocol.SplitRequestPayload{
		URL:          annotation.URL,
		Chunk:        annotation.Chunk,
		SegmentType:  annotation.SegmentType,
		LeftContext:  context,
		RightContext: context,
	}
	res, err := chunkExtractor.ProcessURLWithContext(request, "")
	if err != nil {
		msg := fmt.Sprintf("chunk extractor failed : %v", err)
		jsonError(w, msg, msg)
		return
	}
	chunk := res.Chunk
	res.AnnotationPayload = annotation
	res.Chunk = chunk

	// debug print
	resJSONDbg, _ := res.PrettyMarshal()
	log.Debug("ProcessURLWithContext gave %#v", string(resJSONDbg))

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
	//w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", string(msgJSON))
}

func load(conn *websocket.Conn, annotation protocol.AnnotationPayload, userName string, explicitContext int64) {
	var context int64
	if explicitContext > 0 {
		context = explicitContext
	} else if ctx, ok := contextMap[annotation.SegmentType]; ok {
		context = ctx
	} else {
		context = fallbackContext
	}

	request := protocol.SplitRequestPayload{
		URL:          annotation.URL,
		Chunk:        annotation.Chunk,
		SegmentType:  annotation.SegmentType,
		LeftContext:  context,
		RightContext: context,
	}
	res, err := chunkExtractor.ProcessURLWithContext(request, "")
	if err != nil {
		msg := fmt.Sprintf("chunk extractor failed : %v", err)
		wsError(conn, msg, msg)
		return
	}
	chunk := res.Chunk
	res.AnnotationPayload = annotation
	res.Chunk = chunk

	// debug print
	resJSONDbg, _ := res.PrettyMarshal()
	log.Debug("ProcessURLWithContext gave %#v", string(resJSONDbg))

	wsPayload(conn, "audio_chunk", res)
}

func next(w http.ResponseWriter, r *http.Request) {
	var err error
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		jsonError(w, msg, msg)
		return
	}
	log.Info("next | input: %s", string(body))

	query := protocol.QueryPayload{}
	err = json.Unmarshal(body, &query)
	if err != nil {
		msg := fmt.Sprintf("failed to unmarshal incoming JSON : %v", err)
		jsonError(w, msg, msg)
		return
	}

	log.Info("query %#v", query)
	if query.UserName == "" {
		msg := fmt.Sprintf("User name not provided for query")
		jsonError(w, msg, msg)
		return
	}
	if query.StepSize == 0 {
		msg := fmt.Sprintf("Step size not provided for query")
		jsonError(w, msg, msg)
		return
	}
	if query.CurrID == "undefined" {
		query.CurrID = ""
	}
	sourceFile, err := db.GetNextSegment(query, true)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		jsonError(w, msg, msg)
		return
	}
	load0(sourceFile, query.UserName, query.Context, w)
}

func saveReleaseAndNext(conn *websocket.Conn, payload AnnotationReleaseAndQueryPayload) {
	var err error
	if payload.Annotation.UUID != "" && payload.Annotation.UUID != payload.Release.UUID {
		msg := fmt.Sprintf("mismatching uuids for annotation/release data : %v/%v", payload.Annotation.UUID, payload.Release.UUID)
		wsError(conn, msg, msg)
		return
	}

	// save annotation
	if payload.Annotation.UUID != "" {
		err = db.Save(payload.Annotation)
		if err != nil {
			msg := fmt.Sprintf("failed to save annotation : %v", err)
			wsError(conn, msg, msg)
			return
		}
		msg := fmt.Sprintf("Saved annotation for segment with id %s", payload.Annotation.UUID)
		wsInfo(conn, msg)
	}

	// unlock entry
	if payload.Release.UUID == "" {
		msg := fmt.Sprintf("uuid not provided")
		wsError(conn, msg, msg)
		return
	}
	if payload.Release.UserName == "" {
		msg := fmt.Sprintf("User name not provided for unlock")
		wsError(conn, msg, msg)
		return
	}
	err = db.Unlock(payload.Release.UUID, payload.Release.UserName)
	if err != nil {
		msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
		wsError(conn, msg, msg)
		return
	}
	msg := fmt.Sprintf("Unlocked segment %s for user %s", payload.Release.UUID, payload.Release.UserName)
	wsInfo(conn, msg)

	// get next
	query := payload.Query
	if query.UserName == "" {
		msg := fmt.Sprintf("User name not provided for query")
		wsError(conn, msg, msg)
		return
	}
	if query.StepSize == 0 {
		msg := fmt.Sprintf("Step size not provided for query")
		wsError(conn, msg, msg)
		return
	}
	if query.CurrID == "undefined" {
		query.CurrID = ""
	}
	sourceFile, err := db.GetNextSegment(query, true)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		wsError(conn, msg, msg)
		return
	}
	load(conn, sourceFile, query.UserName, query.Context)
}

func save(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body : %v", err)
		jsonError(w, msg, msg)
		return
	}
	log.Info("save | input: %s", string(body))

	annotation := protocol.AnnotationPayload{}
	err = json.Unmarshal(body, &annotation)
	if err != nil {
		msg := fmt.Sprintf("failed to unmarshal incoming JSON : %v", err)
		jsonError(w, msg, msg)
		return
	}

	err = db.Save(annotation)
	if err != nil {
		msg := fmt.Sprintf("failed to save annotation : %v", err)
		jsonError(w, msg, msg)
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

	r.HandleFunc("/doc/", generateDoc).Methods("GET")
	r.HandleFunc("/ws/{client_id}", wsHandler)

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
