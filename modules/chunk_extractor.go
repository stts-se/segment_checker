package modules

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/google/uuid"

	"github.com/stts-se/TillStud/segment_checker/protocol"
)

// ChunkExtractor is extract time chunks from an audio file, creating a subset of "phrases" from the file.
// For initialization, use NewChunkExtractor().
type ChunkExtractor struct {
	chunk2file Chunk2File
}

// NewChunkExtractor creates a new ChunkExtractor after first checking that the ffmpeg command exists
func NewChunkExtractor() (ChunkExtractor, error) {
	c2f, err := NewChunk2File()
	if err != nil {
		return ChunkExtractor{}, err
	}
	return ChunkExtractor{chunk2file: c2f}, nil
}

// Process the audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) Process(audioFile string, chunks []protocol.TimeChunk, encoding string) ([][]byte, error) {
	res := [][]byte{}
	for _, chunk := range chunks {

		ext := filepath.Ext(audioFile)
		ext = strings.TrimPrefix(ext, ".")
		if encoding != "" {
			ext = encoding
		} else {
			encoding = ext
		}
		id, err := uuid.NewUUID()
		if err != nil {
			return res, fmt.Errorf("couldn't create uuid : %v", err)
		}
		tmpFile := path.Join(os.TempDir(), fmt.Sprintf("chunk-extractor-%s.%s", id, ext))
		//log.Println("chunk_extractor tmpFile", tmpFile)
		defer os.Remove(tmpFile)
		err = ch.chunk2file.ProcessChunk(audioFile, chunk, tmpFile, encoding)
		if err != nil {
			return res, fmt.Errorf("chunk2file.ProcessChunk failed : %v", err)
		}
		bytes, err := ioutil.ReadFile(tmpFile)
		if err != nil {
			return res, fmt.Errorf("failed to read file : %v", err)
		}
		res = append(res, bytes)
	}
	return res, nil
}
