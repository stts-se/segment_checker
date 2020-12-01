package dbapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/stts-se/segment_checker/protocol"
)

type DBAPI struct {
	SourceDataDir, AnnotationDataDir string
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
