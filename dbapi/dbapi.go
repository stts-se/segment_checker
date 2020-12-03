package dbapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/stts-se/segment_checker/protocol"
)

type DBAPI struct {
	SourceDataDir, AnnotationDataDir string
	fileMutex                        *sync.RWMutex     // for file reading/writing
	lockMutex                        *sync.RWMutex     // for segment locking
	lockMap                          map[string]string // segment uuid id -> user
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
		f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", seg.UUID))
		_, err := os.Stat(f)
		if os.IsNotExist(err) && !api.Locked(seg.UUID) {
			res = append(res, seg)
		}
	}
	return res, err
}

func (api *DBAPI) Unlock(uuid, user string) error {
	api.lockMutex.Lock()
	defer api.lockMutex.Unlock()
	lockedBy, exists := api.lockMap[uuid]
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
	delete(api.lockMap, uuid)
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

func (api *DBAPI) Locked(uuid string) bool {
	api.lockMutex.RLock()
	defer api.lockMutex.RUnlock()
	_, res := api.lockMap[uuid]
	return res
}

func (api *DBAPI) Lock(uuid, user string) error {
	api.lockMutex.Lock()
	defer api.lockMutex.Unlock()
	lockedBy, exists := api.lockMap[uuid]
	if exists {
		return fmt.Errorf("%v is already locked by user %s", uuid, lockedBy)
	}
	api.lockMap[uuid] = user
	return nil
}

func (api *DBAPI) CheckedSegmentStats() (int, map[string]int, error) {
	res := map[string]int{}
	all, err := api.ListAllSegments()
	n := 0
	for _, seg := range all {
		f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", seg.UUID))
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

func statusMatch(requestStatus []string, actualStatus string) bool {
	if len(requestStatus) == 0 {
		return true
	}
	for _, req := range requestStatus {
		if req == actualStatus {
			return true
		}
	}
	return false
}

func abs(i int64) int64 {
	if i > 0 {
		return i
	}
	return -i
}

func (api *DBAPI) GetNextSegment(query protocol.QueryPayload, lockOnLoad bool) (protocol.AnnotationPayload, error) {
	api.fileMutex.RLock()
	defer api.fileMutex.RUnlock()
	files, err := ioutil.ReadDir(api.SourceDataDir)
	if err != nil {
		return protocol.AnnotationPayload{}, fmt.Errorf("couldn't list files in folder %s : %v", api.SourceDataDir, err)
	}
	currIndex := 0
	for i, sourceFile := range files {
		if strings.HasSuffix(sourceFile.Name(), ".json") {
			bts, err := ioutil.ReadFile(path.Join(api.SourceDataDir, sourceFile.Name()))
			if err != nil {
				return protocol.AnnotationPayload{}, fmt.Errorf("couldn't read file %s : %v", sourceFile.Name(), err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return protocol.AnnotationPayload{}, fmt.Errorf("couldn't unmarshal json file %s : %v", sourceFile.Name(), err)
			}
			if segment.UUID == query.CurrID {
				currIndex = i
			}
		}
	}
	seenCurrID := int64(-1)
	if query.CurrID == "" {
		seenCurrID = 0
	}
	for i := currIndex; i >= 0 && i < len(files); {
		sourceFile := files[i]
		if strings.HasSuffix(sourceFile.Name(), ".json") {
			bts, err := ioutil.ReadFile(path.Join(api.SourceDataDir, sourceFile.Name()))
			if err != nil {
				return protocol.AnnotationPayload{}, fmt.Errorf("couldn't read file %s : %v", sourceFile.Name(), err)
			}
			var segment protocol.SegmentPayload
			err = json.Unmarshal(bts, &segment)
			if err != nil {
				return protocol.AnnotationPayload{}, fmt.Errorf("couldn't unmarshal json file %s : %v", sourceFile.Name(), err)
			}
			if segment.UUID == query.CurrID {
				seenCurrID = 0
				//log.Debug("GetNextSegment index=%v seenCurrID=%v stepSize=%v CURR!", i+1, seenCurrID, query.StepSize)
			} else {
				annotationFile := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", segment.UUID))
				var annotation protocol.AnnotationPayload
				_, err = os.Stat(annotationFile)
				if err == nil {
					bts, err = ioutil.ReadFile(annotationFile)
					if err != nil {
						return protocol.AnnotationPayload{}, fmt.Errorf("couldn't read file %s : %v", annotationFile, err)
					}
					err = json.Unmarshal(bts, &annotation)
					if err != nil {
						return protocol.AnnotationPayload{}, fmt.Errorf("couldn't unmarshal json file %s : %v", path.Base(annotationFile), err)
					}
					//log.Debug("GetNextSegment annotation %v %#v", seenCurrID, annotation)
				} else {
					annotation = protocol.AnnotationPayload{
						SegmentPayload: segment,
						CurrentStatus:  protocol.Status{Name: "unchecked"},
					}
				}
				//log.Debug("GetNextSegment index=%v seenCurrID=%v stepSize=%v status=%v", i+1, seenCurrID, query.StepSize, annotation.CurrentStatus.Name)
				if seenCurrID >= 0 && statusMatch(query.RequestStatus, annotation.CurrentStatus.Name) && !api.Locked(segment.UUID) {
					seenCurrID++
					if query.CurrID == "" || seenCurrID == abs(query.StepSize) {
						if lockOnLoad {
							api.Lock(annotation.UUID, query.UserName)
						}
						annotation.Index = int64(i + 1)
						return annotation, nil
					}
				}
			}
		}
		if query.StepSize < 0 {
			i--
		} else {
			i++
		}
	}
	return protocol.AnnotationPayload{}, fmt.Errorf("couldn't find any segments to check")
}

func (api *DBAPI) Save(annotation protocol.AnnotationPayload) error {
	f := path.Join(api.AnnotationDataDir, fmt.Sprintf("%s.json", annotation.UUID))
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
