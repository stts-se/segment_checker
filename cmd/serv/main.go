package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/rsc/getopt"

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

type AnnotationUnlockAndQueryPayload struct {
	Annotation protocol.AnnotationPayload `json:"annotation"`
	Unlock     protocol.UnlockPayload     `json:"unlock"`
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
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
		log.Error(msg)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, resJSON)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
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
		msg := fmt.Sprintf("Failed to marshal result : %v", err)
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
		msg := "Expected client ID, got empty string"
		jsonError(w, msg, msg)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to upgrade HTTP request to websocket : %v", err)
		jsonError(w, msg, msg)
	}

	clientMutex.Lock()
	clients[clientID] = ws
	clientMutex.Unlock()
	log.Info("Added websocket for client id %s", clientID)

	// listen forever
	go listenToClient(ws, clientID)
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
		msg := fmt.Sprintf("Failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, jsnMsg)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

func wsInfo(conn *websocket.Conn, msg string) {
	resp := Message{
		//ClientID:    msg.ClientID,
		Info: msg,
	}

	jsnMsg, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal struct into JSON : %v", err)
		wsError(conn, msg, msg)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, jsnMsg)
	if err != nil {
		log.Error("Couldn't write to conn: %v", err)
	}
}

func listenToClient(conn *websocket.Conn, clientID string) {
	//wsInfo(conn, "Websocket created on server")

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			msg := fmt.Sprintf("Websocket error : %v", err)
			log.Error(msg)
			clientMutex.Lock()
			delete(clients, clientID)
			clientMutex.Unlock()
			log.Info("Removed websocket for client id %s", clientID)

			return
		}

		log.Info("Payload received over websocket: %#v\n", msg)

		switch msg.MessageType {
		case "stats":
			res, err := db.Stats()
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}
			wsPayload(conn, "stats", res)

		case "saveunlockandnext":
			var payload AnnotationUnlockAndQueryPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}
			saveUnlockAndNext(conn, payload)
			pushStats()

		case "unlock":
			var payload protocol.UnlockPayload
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
				wsError(conn, msg, msg)
				return
			}

			err = db.Unlock(payload.SegmentID, payload.UserName)
			if err != nil {
				msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
				wsError(conn, msg, msg)
				return
			}
			msg := fmt.Sprintf("Unlocked segment %s for user %s", payload.SegmentID, payload.UserName)
			wsPayload(conn, "explicit_unlock_completed", msg)
			pushStats()

		case "unlock_all":
			var payload protocol.UnlockPayload
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
			msg := fmt.Sprintf("Unlocked %d segment%s for user %s", n, pluralS(n), payload.UserName)
			wsPayload(conn, "explicit_unlock_completed", msg)
			pushStats()

		default:
			log.Error("Unknown message type: %s", msg.MessageType)
		}
	}
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pushStats() {
	res, err := db.Stats()
	if err != nil {
		msg := fmt.Sprintf("Failed to unmarshal payload : %v", err)
		log.Error(msg)
		return
	}
	for _, conn := range clients {
		wsPayload(conn, "stats", res)
	}
	log.Info("Pushed stats to %d client%s", len(clients), pluralS(len(clients)))
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

func load(conn *websocket.Conn, annotation protocol.AnnotationPayload, explicitContext int64) {
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
		msg := fmt.Sprintf("Chunk extractor failed : %v", err)
		wsError(conn, msg, msg)
		return
	}
	chunk := res.Chunk
	res.AnnotationPayload = annotation
	res.Chunk = chunk

	// debug print
	// resJSONDbg, _ := res.PrettyMarshal()
	// log.Debug("ProcessURLWithContext gave %#v", string(resJSONDbg))

	wsPayload(conn, "audio_chunk", res)
}

func saveUnlockAndNext(conn *websocket.Conn, payload AnnotationUnlockAndQueryPayload) {
	var err error
	if payload.Annotation.ID != "" && payload.Annotation.ID != payload.Unlock.SegmentID {
		msg := fmt.Sprintf("Mismatching uuids for annotation/unlock data : %v/%v", payload.Annotation.ID, payload.Unlock.SegmentID)
		wsError(conn, msg, msg)
		return
	}

	log.Info("saveUnlockAndNext | %#v", payload)

	var savedAnnotation protocol.AnnotationPayload

	// save annotation
	if payload.Annotation.ID != "" {
		err = db.Save(payload.Annotation)
		if err != nil {
			msg := fmt.Sprintf("Failed to save annotation : %v", err)
			wsError(conn, msg, msg)
			return
		}
		log.Info("Saved annotation %#v", payload.Annotation)
		msg := fmt.Sprintf("Saved annotation for segment with id %s", payload.Annotation.ID)
		wsInfo(conn, msg)
		savedAnnotation = payload.Annotation
	}

	// get next
	query := payload.Query
	if query.UserName == "" {
		msg := fmt.Sprintf("User name not provided for query")
		wsError(conn, msg, msg)
		return
	}
	if query.StepSize == 0 && query.RequestIndex == "" {
		msg := fmt.Sprintf("Neither step size nor request index was provided for query")
		wsError(conn, msg, msg)
		return
	}
	if query.CurrID == "undefined" {
		query.CurrID = ""
	}
	segment, found, err := db.GetNextSegment(query, true)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		wsError(conn, msg, msg)
		return
	}
	if found {
		load(conn, segment, query.Context)
	} else {
		if query.RequestIndex != "" {
			msg := fmt.Sprintf("Couldn't go to %s segment", query.RequestIndex)
			wsPayload(conn, "no_audio_chunk", msg)
		} else {
			direction := "next"
			if query.StepSize < 0 {
				direction = "previous"
			}
			msg := fmt.Sprintf("Couldn't find any %s segments matching request status: %v", direction, query.RequestStatus)
			wsPayload(conn, "no_audio_chunk", msg)
		}
		if savedAnnotation.ID != "" {
			load(conn, savedAnnotation, query.Context)
		}
		return
	}

	// unlock entry
	if payload.Unlock.SegmentID != "" {
		if payload.Unlock.UserName == "" {
			msg := fmt.Sprintf("User name not provided for unlock")
			wsError(conn, msg, msg)
			return
		}
		err = db.Unlock(payload.Unlock.SegmentID, payload.Unlock.UserName)
		if err != nil {
			msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
			wsError(conn, msg, msg)
			return
		}
		msg := fmt.Sprintf("Unlocked segment %s for user %s", payload.Unlock.SegmentID, payload.Unlock.UserName)
		wsInfo(conn, msg)
	}
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
	cfg.Host = flag.String("host", "localhost", "Server `host`")
	cfg.Port = flag.String("port", "7371", "Server `port`")
	cfg.ServeDir = flag.String("serve", "static", "Serve static `folder`")
	cfg.SourceDataDir = flag.String("source", "", "Source data `folder`")
	cfg.AnnotationDataDir = flag.String("annotation", "", "Annotation data `folder`")

	cfg.Debug = flag.Bool("debug", false, "Debug mode")
	protocol := "http"

	// Shorthand aliases
	getopt.Aliases(
		"h", "host",
		"p", "port",
	)

	help := flag.Bool("help", false, "Print usage and exit")
	flag.Parse()

	cfgJSON, _ := json.Marshal(cfg)
	log.Info("Server config: %#v", string(cfgJSON))

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <options> <folder to serve>\n", cmd)
		fmt.Fprintf(os.Stderr, "Options:\n")
		getopt.PrintDefaults()
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
		msg := fmt.Sprintf("Failure to walk URLs : %v", err)
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
