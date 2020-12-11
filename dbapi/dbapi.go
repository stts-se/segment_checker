package dbapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/stts-se/segment_checker/log"
	"github.com/stts-se/segment_checker/protocol"
)

const (
	debug         = false
	testURLAccess = true
)

type DBAPI struct {
	ProjectDir, SourceDataDir, AnnotationDataDir string

	dbMutex        *sync.RWMutex // for db read/write (files and in-memory saves)
	sourceData     []protocol.SegmentPayload
	annotationData map[string]protocol.AnnotationPayload

	lockMapMutex *sync.RWMutex     // for segment locking
	lockMap      map[string]string // segment id -> user
}

func NewDBAPI(projectDir string) *DBAPI {
	res := DBAPI{
		ProjectDir:        projectDir,
		SourceDataDir:     path.Join(projectDir, "source"),
		AnnotationDataDir: path.Join(projectDir, "annotation"),

		dbMutex:        &sync.RWMutex{},
		sourceData:     []protocol.SegmentPayload{},
		annotationData: map[string]protocol.AnnotationPayload{},

		lockMapMutex: &sync.RWMutex{},
		lockMap:      map[string]string{},
	}
	return &res
}

func (api *DBAPI) LoadData() error {
	var err error

	if api.ProjectDir == "" {
		return fmt.Errorf("project dir not provided")
	}
	if api.SourceDataDir == "" {
		return fmt.Errorf("source dir not set")
	}
	if api.AnnotationDataDir == "" {
		return fmt.Errorf("annotation dir not set")
	}

	info, err := os.Stat(api.ProjectDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("project dir does not exist: %s", api.ProjectDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("project dir is not a directory: %s", api.ProjectDir)
	}

	info, err = os.Stat(api.SourceDataDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("source dir does not exist: %s", api.SourceDataDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("source dir is not a directory: %s", api.SourceDataDir)
	}

	info, err = os.Stat(api.AnnotationDataDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(api.AnnotationDataDir, 0700)
		if err != nil {
			return fmt.Errorf("failed to create annotation folder %s : %v", api.AnnotationDataDir, err)
		}
		log.Info("dbapi Created annotation dir %s", api.AnnotationDataDir)
	} else if !info.IsDir() {
		return fmt.Errorf("annotation dir is not a directory: %s", api.AnnotationDataDir)
	}

	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()
	api.sourceData, err = api.LoadSourceData()
	if err != nil {
		return err
	}
	log.Info("dbapi Loaded %d source files", len(api.sourceData))

	api.annotationData, err = api.LoadAnnotationData()
	if err != nil {
		return err
	}
	log.Info("dbapi Loaded %d annotation files", len(api.annotationData))

	err = api.validateData()
	if err != nil {
		return fmt.Errorf("data validation failed : %v", err)
	}
	log.Info("dbapi Data validated without errors")

	return nil
}

func (api *DBAPI) listJSONFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(pth string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if filepath.Ext(f.Name()) == ".json" {
				files = append(files, path.Join(dir, f.Name()))
			}
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i] < files[j] })
	return files
}

func (api *DBAPI) validateData() error {
	sourceMap := map[string]protocol.SegmentPayload{}
	for _, seg := range api.sourceData {
		sourceMap[seg.ID] = seg
	}
	if len(sourceMap) == 0 {
		return fmt.Errorf("found no segments in source data")
	}
	for id, anno := range api.annotationData {
		seg, segExists := sourceMap[id]
		if !segExists {
			return fmt.Errorf("annotation data with id %s not found in source data", id)
		}
		if anno.URL != seg.URL {
			return fmt.Errorf("annotation data has a different URL than source data: %s vs %s", anno.URL, seg.URL)
		}
		if anno.ID != seg.ID {
			return fmt.Errorf("annotation data has a different ID than source data: %s vs %s", anno.ID, seg.ID)
		}
		if anno.SegmentType != seg.SegmentType {
			return fmt.Errorf("annotation data has a different segment type than source data: %s vs %s", anno.SegmentType, seg.SegmentType)
		}
	}
	return nil
}

func validateSegment(segment protocol.SegmentPayload) error {
	if segment.ID == "" {
		return fmt.Errorf("no id")
	}

	if segment.SegmentType == "" {
		return fmt.Errorf("no segment type")
	}
	if segment.Chunk.Start > segment.Chunk.End {
		return fmt.Errorf("chunk end must be after chunk start, found chunk %#v", segment.Chunk)
	}

	// URL
	if segment.URL == "" {
		return fmt.Errorf("no URL")
	}
	if testURLAccess {
		urlResp, err := http.Get(segment.URL)
		if err != nil {
			return fmt.Errorf("audio URL %s not reachable : %v", segment.URL, err)
		}
		defer urlResp.Body.Close()
		if urlResp.StatusCode != http.StatusOK {
			return fmt.Errorf("audio URL %s not reachable (status %s)", segment.URL, urlResp.Status)
		}
	}

	return nil
}

func validateAnnotation(anno protocol.AnnotationPayload) error {
	if anno.ID == "" {
		return fmt.Errorf("no id")
	}
	if anno.URL == "" {
		return fmt.Errorf("no URL")
	}
	if anno.SegmentType == "" {
		return fmt.Errorf("no anno type")
	}
	if anno.Chunk.Start > anno.Chunk.End {
		return fmt.Errorf("chunk end must be after chunk start, found chunk %#v", anno.Chunk)
	}
	if len(anno.StatusHistory) > 0 && anno.CurrentStatus.Name == "" {
		return fmt.Errorf("status history exists, but no current status: %#v", anno)
	}
	return nil
}

func (api *DBAPI) LoadSourceData() ([]protocol.SegmentPayload, error) {
	res := []protocol.SegmentPayload{}
	files := api.listJSONFiles(api.SourceDataDir)
	seenIDs := make(map[string]bool)
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			bts, err := ioutil.ReadFile(f)
			if err != nil {
				return res, fmt.Errorf("couldn't read segment file %s : %v", f, err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshal segment file %s: %v", f, err)
			}
			err = validateSegment(segment)
			if err != nil {
				return res, fmt.Errorf("invalid segment file %s : %v", f, err)
			}
			if _, seen := seenIDs[segment.ID]; seen {
				return res, fmt.Errorf("duplicate ids for source data: %s", segment.ID)
			}
			seenIDs[segment.ID] = true
			res = append(res, segment)
		}
	}
	return res, nil
}

func (api *DBAPI) LoadAnnotationData() (map[string]protocol.AnnotationPayload, error) {
	res := map[string]protocol.AnnotationPayload{}
	files := api.listJSONFiles(api.AnnotationDataDir)
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			bts, err := ioutil.ReadFile(f)
			if err != nil {
				return res, fmt.Errorf("couldn't read annotation file %s : %v", f, err)
			}
			var annotation protocol.AnnotationPayload
			err = json.Unmarshal(bts, &annotation)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshal annotation file %s : %v", f, err)
			}
			err = validateAnnotation(annotation)
			if err != nil {
				return res, fmt.Errorf("invalid json in annotation file %s : %v", f, err)
			}
			if _, seen := res[annotation.ID]; seen {
				return res, fmt.Errorf("duplicate ids for annotation data: %s", annotation.ID)
			}
			res[annotation.ID] = annotation
		}
	}
	return res, nil
}

func (api *DBAPI) ListUncheckedSegments() []protocol.SegmentPayload {
	res := []protocol.SegmentPayload{}
	for _, seg := range api.sourceData {
		if _, annoExists := api.annotationData[seg.ID]; !annoExists {
			res = append(res, seg)
		}
	}
	return res
}

func (api *DBAPI) Unlock(segmentID, user string) error {
	log.Info("dbapi Unlock %s %s", segmentID, user)
	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
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
	api.lockMapMutex.RLock()
	defer api.lockMapMutex.RUnlock()
	_, res := api.lockMap[segmentID]
	return res
}

func (api *DBAPI) Lock(segmentID, user string) error {
	log.Info("dbapi Lock %s %s", segmentID, user)
	api.lockMapMutex.Lock()
	defer api.lockMapMutex.Unlock()
	lockedBy, exists := api.lockMap[segmentID]
	if exists {
		return fmt.Errorf("%v is already locked by user %s", segmentID, lockedBy)
	}
	api.lockMap[segmentID] = user
	return nil
}

func (api *DBAPI) CheckedSegmentStats() (int, map[string]int) {
	res := map[string]int{}
	for _, anno := range api.annotationData {
		badSample := false
		for _, l := range anno.Labels {
			if l == StatusBadSample {
				badSample = true
			}
		}
		status := anno.CurrentStatus
		if badSample {
			res["status:bad sample"]++
		} else {
			res["status:"+status.Name]++
		}
		if len(status.Source) > 0 {
			res["checked by:"+status.Source]++
		}
		if strings.TrimSpace(anno.Comment) != "" {
			res["comment"]++
		}
	}
	return len(api.annotationData), res
}

func (api *DBAPI) Stats() (map[string]int, error) {
	allSegs, err := api.LoadSourceData()
	if err != nil {
		return map[string]int{}, fmt.Errorf("couldn't list segments: %v", err)
	}
	checkableSegs := api.ListUncheckedSegments()
	nChecked, checkedStats := api.CheckedSegmentStats()
	res := map[string]int{
		"total":     len(allSegs),
		"checked":   nChecked,
		"unchecked": len(checkableSegs),
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
	StatusBadSample = "bad sample"

	StatusChecked = "checked"
	StatusAny     = "any"
	StatusEmpty   = ""
)

func statusMatch(requestStatus string, actualStatus string, labels []string) bool {
	badSample := false
	for _, l := range labels {
		if l == StatusBadSample {
			badSample = true
		}
	}
	switch requestStatus {
	case StatusChecked:
		return actualStatus != StatusUnchecked && actualStatus != StatusEmpty
	case StatusAny:
		return true
	case StatusBadSample:
		return badSample
	case StatusSkip:
		return !badSample && actualStatus == StatusSkip
	default:
		return actualStatus == requestStatus
	}
	return false
}

func abs(i int64) int64 {
	if i > 0 {
		return i
	}
	return -i
}

func (api *DBAPI) annotationFromSegment(segment protocol.SegmentPayload) protocol.AnnotationPayload {
	annotation, exists := api.annotationData[segment.ID]
	if exists {
		return annotation
	}
	return protocol.AnnotationPayload{
		SegmentPayload: segment,
		CurrentStatus:  protocol.Status{Name: "unchecked"},
	}
}

func (api *DBAPI) GetNextSegment(query protocol.QueryPayload, lockOnLoad bool) (protocol.AnnotationPayload, bool, error) {
	log.Info("dbapi GetNextSegment")
	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()

	if debug {
		log.Debug("dbapi GetNextSegment query: %#v", query)
	}

	var currIndex int
	var seenCurrID int64
	if query.RequestIndex != "" {
		var i int
		if query.RequestIndex == "first" {
			i = 0
		} else if query.RequestIndex == "last" {
			i = len(api.sourceData) - 1
		} else {
			reqI, err := strconv.Atoi(query.RequestIndex)
			if err == nil && reqI >= 0 && reqI < len(api.sourceData) {
				i = reqI
			} else {
				return protocol.AnnotationPayload{}, false, fmt.Errorf("invalid request index: %s", query.RequestIndex)
			}
		}
		segment := api.sourceData[i]
		annotation := api.annotationFromSegment(segment)
		if lockOnLoad {
			api.Lock(annotation.ID, query.UserName)
		}
		annotation.Index = int64(i + 1)
		return annotation, true, nil
	} else if query.CurrID != "" {
		seenCurrID = int64(-1)
		for i, segment := range api.sourceData {
			if segment.ID == query.CurrID {
				currIndex = i
			}
		}
	} else {
		seenCurrID = int64(0)
		currIndex = 0
	}
	for i := currIndex; i >= 0 && i < len(api.sourceData); {
		segment := api.sourceData[i]
		if seenCurrID < 0 && segment.ID == query.CurrID {
			seenCurrID = 0
			if debug {
				log.Debug("dbapi GetNextSegment index=%v seenCurrID=%v segment.ID=%v stepSize=%v CURR!", i+1, seenCurrID, segment.ID, query.StepSize)
			}
		} else {
			annotation := api.annotationFromSegment(segment)
			if debug {
				log.Debug("dbapi GetNextSegment index=%v seenCurrID=%v segment.ID=%v stepSize=%v status=%v", i+1, seenCurrID, segment.ID, query.StepSize, annotation.CurrentStatus.Name)
			}
			if seenCurrID >= 0 && statusMatch(query.RequestStatus, annotation.CurrentStatus.Name, annotation.Labels) && !api.Locked(segment.ID) {
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
	log.Info("dbapi Save %#v", annotation)

	api.dbMutex.RLock()
	defer api.dbMutex.RUnlock()

	/* SAVE TO CACHE */
	api.annotationData[annotation.ID] = annotation

	/* PRINT TO FILE */

	// create copy for writing, and remove internal index
	saveAnno := annotation
	saveAnno.Index = 0

	f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", annotation.ID))
	writeJSON, err := json.MarshalIndent(saveAnno, " ", " ")
	if err != nil {
		return fmt.Errorf("marhsal failed : %v", err)
	}

	file, err := os.Create(f)
	if err != nil {
		return fmt.Errorf("failed create file %s : %v", f, err)
	}
	defer file.Close()
	file.Write(writeJSON)

	return nil
}
