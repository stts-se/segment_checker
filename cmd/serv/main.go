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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

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

func load0(sourceFile, userName string, context int64, w http.ResponseWriter) {
	fileMutex.RLock()
	defer fileMutex.RUnlock()
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
	err = lock(segment.UUID, userName)
	if err != nil {
		msg := fmt.Sprintf("Couldn't lock segment: %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	request := protocol.SplitRequestPayload{
		URL:          segment.URL,
		Chunk:        segment.Chunk,
		SegmentType:  segment.SegmentType,
		LeftContext:  context,
		RightContext: context,
	}
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
		httpError(w, msg, msg, http.StatusBadRequest)
		return

	}
	msg := Message{
		MessageType: "audio_chunk",
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
	context := int64(1000)
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
	userName := getParam("username", r)
	currID := getParam("currid", r)
	contextS := getParam("context", r)
	if userName == "" {
		msg := fmt.Sprintf("User name not provided")
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	if currID == "undefined" {
		currID = ""
	}
	context := int64(1000)
	if contextS != "" {
		context, err = strconv.ParseInt(contextS, 10, 64)
		if err != nil {
			msg := fmt.Sprintf("Couldn't parse int %s", contextS)
			jsonError(w, msg, msg, http.StatusBadRequest)
			return
		}
	}

	log.Info("next | input: %s %s %v", userName, currID, contextS)
	sourceFile, err := getNextCheckableSegment(currID)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	load0(sourceFile, userName, context, w)
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
	log.Info("load | input: %s %s", uuid, userName)

	err := unlock(uuid, userName)
	if err != nil {
		msg := fmt.Sprintf("Couldn't unlock segment: %v", err)
		jsonError(w, msg, msg, http.StatusBadRequest)
		return
	}
	msg := Message{Info: fmt.Sprintf("Unlocked segment %s for user %s", uuid, userName)}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
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

	n := 0
	for k, v := range lockMap {
		if v == userName {
			unlock(k, v)
			n++
		}
	}

	msg := Message{Info: fmt.Sprintf("Unlocked %d segments for user %s", n, userName)}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal msg : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
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
	f := path.Join(*cfg.AnnotationDataDir, fmt.Sprintf("%s.json", annotation.UUID))
	writeJSON, err := json.MarshalIndent(annotation, " ", " ")
	if err != nil {
		msg := fmt.Sprintf("Marshal failed: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}

	fileMutex.Lock()
	defer fileMutex.Unlock()
	file, err := os.Create(f)
	if err != nil {
		msg := fmt.Sprintf("Couldn't create file %s: %v", f, err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	defer file.Close()
	file.Write(writeJSON)

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

func locked(uuid string) bool {
	lockMutex.RLock()
	defer lockMutex.RUnlock()
	_, res := lockMap[uuid]
	return res
}

func lock(uuid, user string) error {
	lockMutex.Lock()
	defer lockMutex.Unlock()
	lockedBy, exists := lockMap[uuid]
	if exists {
		return fmt.Errorf("%v is already locked by user %s", uuid, lockedBy)
	}
	lockMap[uuid] = user
	return nil
}

func unlock(uuid, user string) error {
	lockMutex.Lock()
	defer lockMutex.Unlock()
	lockedBy, exists := lockMap[uuid]
	if !exists {
		//log.Warning("unlock: %v is not locked", uuid)
		return fmt.Errorf("%v is not locked", uuid)
		//return nil
	}
	if lockedBy != user {
		//log.Warning("unlock: %v is not locked by user %s", uuid, user)
		return fmt.Errorf("%v is not locked by user %s", uuid, user)
		//return nil
	}
	delete(lockMap, uuid)
	return nil
}

func stats(w http.ResponseWriter, r *http.Request) {
	allSegs, err := listAllSegments()
	if err != nil {
		msg := fmt.Sprintf("Couldn't list segments: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	checkableSegs, err := listCheckableSegments()
	if err != nil {
		msg := fmt.Sprintf("Couldn't list checkable segments: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	nChecked, checkedStats, err := checkedSegmentStats()
	if err != nil {
		msg := fmt.Sprintf("Couldn't list checked segments: %v", err)
		jsonError(w, msg, msg, http.StatusInternalServerError)
		return
	}
	res := map[string]int{
		"total":     len(allSegs),
		"checked":   nChecked,
		"checkable": len(checkableSegs),
		"locked":    len(lockMap),
	}
	for label, count := range checkedStats {
		res[label] = count
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal result : %v", err)
		httpError(w, msg, msg, http.StatusBadRequest)
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

func getNextCheckableSegment(currID string) (string, error) {
	fileMutex.RLock()
	defer fileMutex.RUnlock()
	files, err := ioutil.ReadDir(*cfg.SourceDataDir)
	if err != nil {
		return "", fmt.Errorf("couldn't list files in folder %s : %v", *cfg.SourceDataDir, err)
	}
	seenCurrID := false
	fallbackFile := ""
	for _, sourceFile := range files {
		if strings.HasSuffix(sourceFile.Name(), ".json") {
			bts, err := ioutil.ReadFile(path.Join(*cfg.SourceDataDir, sourceFile.Name()))
			if err != nil {
				return "", fmt.Errorf("couldn't read file %s : %v", sourceFile.Name(), err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return "", fmt.Errorf("couldn't unmarshal json : %v", err)
			}
			if segment.UUID == currID {
				seenCurrID = true
				continue
			}
			annotationFile := path.Join(*cfg.AnnotationDataDir, fmt.Sprintf("%s.json", segment.UUID))
			_, err = os.Stat(annotationFile)
			if os.IsNotExist(err) && !locked(segment.UUID) {
				if currID == "" || seenCurrID {
					return sourceFile.Name(), nil
				}
				fallbackFile = sourceFile.Name()
			}

		}
	}
	if fallbackFile != "" {
		return fallbackFile, nil
	}
	return "", fmt.Errorf("couldn't find any segments to check")
}

func listAllSegments() ([]protocol.SegmentPayload, error) {
	res := []protocol.SegmentPayload{}
	files, err := ioutil.ReadDir(*cfg.SourceDataDir)
	if err != nil {
		return res, fmt.Errorf("couldn't list files in folder %s : %v", *cfg.SourceDataDir, err)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			bts, err := ioutil.ReadFile(path.Join(*cfg.SourceDataDir, f.Name()))
			if err != nil {
				return res, fmt.Errorf("couldn't read file %s : %v", f, err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshal json : %v", err)
			}
			res = append(res, segment)
		}
	}
	return res, nil
}

func listCheckableSegments() ([]protocol.SegmentPayload, error) {
	res := []protocol.SegmentPayload{}
	all, err := listAllSegments()
	for _, seg := range all {
		f := path.Join(*cfg.AnnotationDataDir, fmt.Sprintf("%s.json", seg.UUID))
		_, err := os.Stat(f)
		if os.IsNotExist(err) && !locked(seg.UUID) {
			res = append(res, seg)
		}
	}
	return res, err
}

func checkedSegmentStats() (int, map[string]int, error) {
	res := map[string]int{}
	all, err := listAllSegments()
	n := 0
	for _, seg := range all {
		f := path.Join(*cfg.AnnotationDataDir, fmt.Sprintf("%s.json", seg.UUID))
		_, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue
		}
		bts, err := ioutil.ReadFile(f)
		if err != nil {
			return n, res, fmt.Errorf("couldn't read file %s : %v", f, err)
		}
		var segment protocol.AnnotationPayload
		err = json.Unmarshal(bts, &segment)
		if err != nil {
			return n, res, fmt.Errorf("couldn't unmarshal json : %v", err)
		}
		n++
		if _, ok := res["status:"+segment.Status.Name]; !ok {
			res["status:"+segment.Status.Name] = 0
		}
		res["status:"+segment.Status.Name]++
		if _, ok := res["editor:"+segment.Status.Source]; !ok {
			res["editor:"+segment.Status.Source] = 0
		}
		res["editor:"+segment.Status.Source]++
		for _, label := range segment.Labels {
			if _, ok := res["label:"+label]; !ok {
				res["label:"+label] = 0
			}
			res["label:"+label]++
		}
	}
	return n, res, err
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

func loadData(dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("data dir not provided")
	}
	info, err := os.Stat(dataDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("provided data dir does not exist: %s", dataDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("provided data dir is not a directory: %s", dataDir)
	}
	// todo: load into memory or read from disk?
	return nil
}

var cfg = &Config{}

// Config for server
type Config struct {
	Host              *string `json:"host"`
	Port              *string `json:"port"`
	ServeDir          *string `json:"static_dir"`
	SourceDataDir     *string `json:"source_data_dir"`
	AnnotationDataDir *string `json:"annotation_data_dir"`
	Debug             *bool   `json:"debug"`
}

var fileMutex = sync.RWMutex{}        // for file saving
var lockMutex = sync.RWMutex{}        // for segment locking
var lockMap = make(map[string]string) // segment uuid id -> user

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

	err = loadData(*cfg.SourceDataDir)
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
	r.HandleFunc("/next", next).Methods("GET")
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
