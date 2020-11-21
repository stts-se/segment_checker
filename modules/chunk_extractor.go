package modules

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/stts-se/segment_checker/protocol"
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

func downloadFile(url, fileName string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("received non 200 response code: %v", response.StatusCode)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

// ProcessFileWithContext an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFileWithContext(audioFile string, chunk protocol.Chunk, leftContext, rightContext int64, encoding string) (protocol.AudioChunk, error) {
	start := chunk.Start - leftContext
	if start < 0 {
		start = 0
	}
	end := chunk.End + rightContext
	processChunk := protocol.Chunk{Start: start, End: end}

	ext := filepath.Ext(audioFile)
	ext = strings.TrimPrefix(ext, ".")
	if encoding != "" {
		ext = encoding
	} else {
		encoding = ext
	}

	btss, err := ch.ProcessFile(audioFile, []protocol.Chunk{processChunk}, encoding)
	if err != nil {
		return protocol.AudioChunk{}, err
	}

	if len(btss) != 1 {
		return protocol.AudioChunk{}, fmt.Errorf("expected one byte array, found %d", len(btss))
	}

	bts := btss[0]

	//ioutil.WriteFile("chunk_extractor_debug.wav", bts, 0644)

	res := protocol.AudioChunk{
		Audio:    base64.StdEncoding.EncodeToString(bts),
		FileType: encoding,
		Chunk: protocol.Chunk{
			Start: chunk.Start - start,
			End:   chunk.End - start,
		},
	}
	return res, nil
}

// ProcessURLWithContext an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURLWithContext(payload protocol.SplitRequestPayload, encoding string) (protocol.AudioChunk, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return protocol.AudioChunk{}, fmt.Errorf("couldn't create uuid : %v", err)
	}
	fileName := filepath.Base(payload.URL)
	audioFile := path.Join(os.TempDir(), fmt.Sprintf("%s-%s", id, fileName))
	defer os.Remove(audioFile)

	err = downloadFile(payload.URL, audioFile)
	if err != nil {
		return protocol.AudioChunk{}, fmt.Errorf("couldn't download URL : %v", err)
	}

	return ch.ProcessFileWithContext(audioFile, payload.Chunk, payload.LeftContext, payload.RightContext, encoding)
}

// ProcessURL an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURL(audioURL string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return [][]byte{}, fmt.Errorf("couldn't create uuid : %v", err)
	}
	fileName := filepath.Base(audioURL)
	audioFile := path.Join(os.TempDir(), fmt.Sprintf("%s-%s", id, fileName))
	defer os.Remove(audioFile)

	err = downloadFile(audioURL, audioFile)
	if err != nil {
		return [][]byte{}, fmt.Errorf("couldn't download URL : %v", err)
	}

	return ch.ProcessFile(audioFile, chunks, encoding)
}

// ProcessFile an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFile(audioFile string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
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
