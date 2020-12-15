package ffmpeg

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/stts-se/segment_checker/protocol"
)

// ChunkExtractor extracts time chunks from an audio file, creating a subset of "phrases" from the file.
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

// ProcessFileWithContext an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFileWithContext(audioFile string, chunk protocol.Chunk, leftContext, rightContext int64, encoding string) (protocol.AudioChunk, error) {
	offset := chunk.Start - leftContext
	if offset < 0 {
		offset = 0
	}
	processChunk := protocol.Chunk{
		Start: offset,
		End:   chunk.End + rightContext,
	}

	ext := filepath.Ext(audioFile)
	ext = strings.TrimPrefix(ext, ".")
	if encoding == "" {
		encoding = ext
	}
	// if encoding != "" {
	// 	ext = encoding
	// } else {
	// 	encoding = ext
	// }

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
		Offset:   offset,
	}
	res.Chunk = protocol.Chunk{
		Start: chunk.Start - offset,
		End:   chunk.End - offset,
	}
	return res, nil
}

// ProcessURLWithContext an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURLWithContext(payload protocol.SplitRequestPayload, encoding string) (protocol.AudioChunk, error) {
	return ch.ProcessFileWithContext(payload.URL, payload.Chunk, payload.LeftContext, payload.RightContext, encoding)
}

// ProcessURL an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURL(audioURL string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
	return ch.ProcessFile(audioURL, chunks, encoding)
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
