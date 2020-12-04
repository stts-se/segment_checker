package dbapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/stts-se/segment_checker/log"
	"github.com/stts-se/segment_checker/protocol"
)

const debug = false

type DBAPI struct {
	SourceDataDir, AnnotationDataDir string
	fileMutex                        *sync.RWMutex     // for file reading/writing
	lockMutex                        *sync.RWMutex     // for segment locking
	lockMap                          map[string]string // segment id -> user
}

func NewDBAPI(sourceDataDir, annotationDataDir string) *DBAPI {
	res := DBAPI{
		SourceDataDir:     sourceDataDir,
		AnnotationDataDir: annotationDataDir,
		fileMutex:         &sync.RWMutex{},
		lockMutex:         &sync.RWMutex{},
		lockMap:           map[string]string{},
	}
	return &res
}

func (api *DBAPI) LoadData() error {
	if api.SourceDataDir == "" {
		return fmt.Errorf("source data dir not provided")
	}
	info, err := os.Stat(api.SourceDataDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("provided source data dir does not exist: %s", api.SourceDataDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("provided source data dir is not a directory: %s", api.SourceDataDir)
	}
	// TODO: Load source data into memory + sort!
	// TODO: Load annotation data into memory + sort!
	return nil
}

func (api *DBAPI) ListAllSegments() ([]protocol.SegmentPayload, error) {
	res := []protocol.SegmentPayload{}
	files, err := ioutil.ReadDir(api.SourceDataDir)
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	if err != nil {
		return res, fmt.Errorf("couldn't list files in folder %s : %v", api.SourceDataDir, err)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			bts, err := ioutil.ReadFile(path.Join(api.SourceDataDir, f.Name()))
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

func (api *DBAPI) ListUncheckedSegments() ([]protocol.SegmentPayload, error) {
	res := []protocol.SegmentPayload{}
	all, err := api.ListAllSegments()
	for _, seg := range all {
		f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", seg.ID))
		_, err := os.Stat(f)
		if os.IsNotExist(err) && !api.Locked(seg.ID) {
			res = append(res, seg)
		}
	}
	return res, err
}

func (api *DBAPI) Unlock(segmentID, user string) error {
	api.lockMutex.Lock()
	defer api.lockMutex.Unlock()
	lockedBy, exists := api.lockMap[segmentID]
	if !exists {
		return fmt.Errorf("%v is not locked", segmentID)
	}
	if lockedBy != user {
		return fmt.Errorf("%v is not locked by user %s", segmentID, user)
	}
	delete(api.lockMap, segmentID)
	return nil
}

func (api *DBAPI) UnlockAll(user string) (int, error) {
	n := 0
	for k, v := range api.lockMap {
		if v == user {
			err := api.Unlock(k, v)
			if err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func (api *DBAPI) Locked(segmentID string) bool {
	api.lockMutex.RLock()
	defer api.lockMutex.RUnlock()
	_, res := api.lockMap[segmentID]
	return res
}

func (api *DBAPI) Lock(segmentID, user string) error {
	log.Info("dbapi.Lock %s %s", segmentID, user)
	api.lockMutex.Lock()
	defer api.lockMutex.Unlock()
	lockedBy, exists := api.lockMap[segmentID]
	if exists {
		return fmt.Errorf("%v is already locked by user %s", segmentID, lockedBy)
	}
	api.lockMap[segmentID] = user
	return nil
}

func (api *DBAPI) CheckedSegmentStats() (int, map[string]int, error) {
	res := map[string]int{}
	all, err := api.ListAllSegments()
	n := 0
	for _, seg := range all {
		f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", seg.ID))
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
		status := segment.CurrentStatus

		res["status:"+status.Name]++
		if len(status.Source) > 0 {
			res["checked by:"+status.Source]++
		}
		for _, label := range segment.Labels {
			res["label:"+label]++
		}
	}
	return n, res, err
}

func (api *DBAPI) Stats() (map[string]int, error) {
	allSegs, err := api.ListAllSegments()
	if err != nil {
		return map[string]int{}, fmt.Errorf("couldn't list segments: %v", err)
	}
	checkableSegs, err := api.ListUncheckedSegments()
	if err != nil {
		return map[string]int{}, fmt.Errorf("couldn't list checkable segments: %v", err)
	}
	nChecked, checkedStats, err := api.CheckedSegmentStats()
	if err != nil {
		return map[string]int{}, fmt.Errorf("couldn't list checked segments: %v", err)
	}
	res := map[string]int{
		"total":     len(allSegs),
		"checked":   nChecked,
		"checkable": len(checkableSegs),
		"locked":    len(api.lockMap),
	}
	for label, count := range checkedStats {
		res[label] = count
	}
	for _, user := range api.lockMap {
		res["locked by:"+user]++
	}
	return res, nil
}

const (
	StatusUnchecked = "unchecked"
	StatusSkip      = "skip"
	StatusOK        = "ok"

	StatusChecked = "checked"
	StatusAny     = "any"
	StatusEmpty   = ""
)

func statusMatch(requestStatus string, actualStatus string) bool {
	switch requestStatus {
	case StatusChecked:
		return actualStatus != StatusUnchecked && actualStatus != StatusEmpty
	case StatusAny:
		return true
	default:
		return actualStatus == requestStatus
	}
}

func abs(i int64) int64 {
	if i > 0 {
		return i
	}
	return -i
}

func (api *DBAPI) annotationFromSegment(segment protocol.SegmentPayload) (protocol.AnnotationPayload, error) {
	var err error
	annotationFile := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", segment.ID))
	var annotation protocol.AnnotationPayload
	_, err = os.Stat(annotationFile)
	if err == nil {
		bts, err := ioutil.ReadFile(annotationFile)
		if err != nil {
			return protocol.AnnotationPayload{}, fmt.Errorf("couldn't read file %s : %v", annotationFile, err)
		}
		err = json.Unmarshal(bts, &annotation)
		if err != nil {
			return protocol.AnnotationPayload{}, fmt.Errorf("couldn't unmarshal json file %s : %v", path.Base(annotationFile), err)
		}
	} else {
		annotation = protocol.AnnotationPayload{
			SegmentPayload: segment,
			CurrentStatus:  protocol.Status{Name: "unchecked"},
		}
	}
	return annotation, nil
}

func (api *DBAPI) segmentFromSource(sourceFile string) (protocol.SegmentPayload, error) {
	bts, err := ioutil.ReadFile(path.Join(api.SourceDataDir, sourceFile))
	if err != nil {
		return protocol.SegmentPayload{}, fmt.Errorf("couldn't read file %s : %v", sourceFile, err)
	}
	var segment protocol.SegmentPayload
	err = json.Unmarshal(bts, &segment)
	if err != nil {
		return protocol.SegmentPayload{}, fmt.Errorf("couldn't unmarshal json file %s : %v", sourceFile, err)
	}
	return segment, nil
}

func (api *DBAPI) GetNextSegment(query protocol.QueryPayload, lockOnLoad bool) (protocol.AnnotationPayload, bool, error) {
	api.fileMutex.RLock()
	defer api.fileMutex.RUnlock()

	if debug {
		log.Debug("GetNextSegment query: %#v", query)
	}

	var files []string
	filepath.Walk(api.SourceDataDir, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if filepath.Ext(f.Name()) == ".json" {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	if len(files) == 0 {
		return protocol.AnnotationPayload{}, false, fmt.Errorf("no source files found in folder %s", api.SourceDataDir)
	}
	log.Info("Loaded %d source data files", len(files))
	var currIndex int
	var seenCurrID int64
	if query.RequestIndex != "" {
		var i int
		if query.RequestIndex == "first" {
			i = 0
		} else if query.RequestIndex == "last" {
			i = len(files) - 1
		} else {
			return protocol.AnnotationPayload{}, false, fmt.Errorf("unknown request index: %s", query.RequestIndex)
		}
		sourceFile := files[i]
		segment, err := api.segmentFromSource(sourceFile)
		if err != nil {
			return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't create segment from source file %s : %v", sourceFile, err)
		}
		annotation, err := api.annotationFromSegment(segment)
		if err != nil {
			return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't create annotation from source file %s : %v", sourceFile, err)
		}
		if lockOnLoad {
			api.Lock(annotation.ID, query.UserName)
		}
		annotation.Index = int64(i + 1)
		return annotation, true, nil
	} else if query.CurrID != "" {
		seenCurrID = int64(-1)
		for i, sourceFile := range files {
			bts, err := ioutil.ReadFile(path.Join(api.SourceDataDir, sourceFile))
			if err != nil {
				return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't read file %s : %v", sourceFile, err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't unmarshal json file %s : %v", sourceFile, err)
			}
			if segment.ID == query.CurrID {
				currIndex = i
			}
		}
	} else {
		seenCurrID = int64(0)
		currIndex = 0
	}
	for i := currIndex; i >= 0 && i < len(files); {
		sourceFile := files[i]
		segment, err := api.segmentFromSource(sourceFile)
		if err != nil {
			return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't create segment from source file %s : %v", sourceFile, err)
		}

		if seenCurrID < 0 && segment.ID == query.CurrID {
			seenCurrID = 0
			if debug {
				log.Debug("GetNextSegment index=%v seenCurrID=%v segment.ID=%v stepSize=%v CURR!", i+1, seenCurrID, segment.ID, query.StepSize)
			}
		} else {
			annotation, err := api.annotationFromSegment(segment)
			if err != nil {
				return protocol.AnnotationPayload{}, false, fmt.Errorf("couldn't create annotation from source file %s : %v", sourceFile, err)
			}
			if debug {
				log.Debug("GetNextSegment index=%v seenCurrID=%v segment.ID=%v stepSize=%v status=%v", i+1, seenCurrID, segment.ID, query.StepSize, annotation.CurrentStatus.Name)
			}
			if seenCurrID >= 0 && statusMatch(query.RequestStatus, annotation.CurrentStatus.Name) && !api.Locked(segment.ID) {
				seenCurrID++
				if query.CurrID == "" || seenCurrID == abs(query.StepSize) {
					if lockOnLoad {
						api.Lock(annotation.ID, query.UserName)
					}
					annotation.Index = int64(i + 1)
					return annotation, true, nil
				}
			}
		}
		if query.StepSize < 0 {
			i--
		} else {
			i++
		}
	}
	return protocol.AnnotationPayload{}, false, nil
}

func (api *DBAPI) Save(annotation protocol.AnnotationPayload) error {
	f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", annotation.ID))
	writeJSON, err := json.MarshalIndent(annotation, " ", " ")
	if err != nil {
		return fmt.Errorf("marhsal failed : %v", err)
	}

	api.fileMutex.Lock()
	defer api.fileMutex.Unlock()
	file, err := os.Create(f)
	if err != nil {
		return fmt.Errorf("failed create file %s : %v", f, err)
	}
	defer file.Close()
	file.Write(writeJSON)
	return nil
}
