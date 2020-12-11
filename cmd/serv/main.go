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

	//	"github.com/rsc/getopt"

	"github.com/stts-se/segment_checker/dbapi"
	"github.com/stts-se/segment_checker/log"
	"github.com/stts-se/segment_checker/modules"
	"github.com/stts-se/segment_checker/protocol"
)

type ClientID struct {
	ID       string `json:"id"`
	UserName string `json:"user_name"`
}

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
var clients = make(map[ClientID]*websocket.Conn)

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
	userName := vars["user_name"]

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to upgrade HTTP request to websocket : %v", err)
		jsonError(w, msg, msg)
		return
	}

	for clID := range clients {
		if clID.UserName == userName {
			msg := fmt.Sprintf("User %s is already logged in", userName)
			wsError(ws, msg, msg)
			ws.Close()
			return
		}
	}

	clID := ClientID{ID: clientID, UserName: userName}
	clientMutex.Lock()
	clients[clID] = ws
	clientMutex.Unlock()
	log.Info("Added websocket for client id %s", clID)

	// listen forever
	go listenToClient(ws, clID)
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

func listenToClient(conn *websocket.Conn, clientID ClientID) {
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

		//log.Info("Payload received over websocket: %#v\n", msg)

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
		serverMsg := fmt.Sprintf("Chunk extractor failed : %v", err)
		wsError(conn, serverMsg, fmt.Sprintf("Chunk extractor failed for %s. See server log for details.", request.URL))
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
	segment, msg, err := db.GetNextSegment(query, payload.Unlock.SegmentID, true)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		wsError(conn, msg, msg)
		return
	}
	if msg == "" || segment.ID != "" {
		load(conn, segment, query.Context)
	} else {
		msgFmted := ""
		if msg != "" {
			msgFmted = fmt.Sprintf(": %s", msg)
		}
		if query.RequestIndex != "" {
			msg := fmt.Sprintf("Couldn't go to %s segment%s", query.RequestIndex, msgFmted)
			wsPayload(conn, "no_audio_chunk", msg)
		} else {
			direction := "next"
			if query.StepSize < 0 {
				direction = "previous"
			}
			msg := fmt.Sprintf("Couldn't find any %s segments matching status %v%s", direction, query.RequestStatus, msgFmted)
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

func serveAudio(w http.ResponseWriter, r *http.Request) {
	file := getParam("file", r)
	http.ServeFile(w, r, path.Join(*cfg.ProjectDir, "audio", file))
}

var cfg = &Config{}
var db *dbapi.DBAPI

// Config for server
type Config struct {
	Host       *string `json:"host"`
	Port       *string `json:"port"`
	ServeDir   *string `json:"static_dir"`
	BlockAudio *bool   `json:"block_audio"`
	ProjectDir *string `json:"project_dir"`
	Debug      *bool   `json:"debug"`
	Ffmpeg     *string `json:"ffmpeg"`
}

func main() {
	var err error

	cmd := path.Base(os.Args[0])

	// Flags
	cfg.Host = flag.String("host", "localhost", "Server `host`")
	cfg.Port = flag.String("port", "7371", "Server `port`")
	cfg.ServeDir = flag.String("serve", "static", "Serve static `folder`")
	cfg.BlockAudio = flag.Bool("block_audio", false, "Block audio folder from being served")
	cfg.ProjectDir = flag.String("project", "", "Project `folder`")
	cfg.Ffmpeg = flag.String("ffmpeg", "ffmpeg", "Ffmpeg command/path")

	cfg.Debug = flag.Bool("debug", false, "Debug mode")
	protocol := "http"

	help := flag.Bool("help", false, "Print usage and exit")
	flag.Parse()

	cfgJSON, _ := json.Marshal(cfg)
	log.Info("Server config: %#v", string(cfgJSON))

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <flags>\n", cmd)
		fmt.Fprintf(os.Stderr, "Flags:\n")
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

	if *cfg.ProjectDir == "" {
		fmt.Fprintf(os.Stderr, "Required flag project not set\n")
		flag.Usage()
		os.Exit(1)
	}

	_, err = os.Stat(*cfg.ServeDir)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Required flag serve points to a non-existing folder: %s\n", *cfg.ServeDir)
		flag.Usage()
		os.Exit(1)
	}

	db = dbapi.NewDBAPI(*cfg.ProjectDir)

	modules.FfmpegCmd = *cfg.Ffmpeg
	chunkExtractor, err = modules.NewChunkExtractor()
	if err != nil {
		log.Fatal("Couldn't initialize chunk extractor: %v", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/doc/", generateDoc).Methods("GET")
	r.HandleFunc("/ws/{client_id}/{user_name}", wsHandler)
	if !*cfg.BlockAudio {
		r.HandleFunc("/audio/{file}", serveAudio).Methods("GET")
	}

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
		log.Fatal(msg)
	}

	info, err := os.Stat(*cfg.ServeDir)
	if os.IsNotExist(err) {
		log.Fatal("Serve dir %s does not exist", *cfg.ServeDir)
	}
	if !info.IsDir() {
		log.Fatal("Serve dir %s is not a directory", *cfg.ServeDir)
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(*cfg.ServeDir))))

	srv := &http.Server{
		Handler:      r,
		Addr:         *cfg.Host + ":" + *cfg.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Info("Starting server on %s://%s", protocol, *cfg.Host+":"+*cfg.Port)
	log.Info("Serving folder %s", *cfg.ServeDir)

	go func() {
		// wait for the server to start, and then load data, including URL access tests
		// (which won't work if it's run before the server is started)
		time.Sleep(1000)
		err = db.LoadData()
		if err != nil {
			log.Fatal("Couldn't load data: %v", err)
		}
	}()

	if err = srv.ListenAndServe(); err != nil {
		log.Fatal("Server failure: %v", err)
	}

}
