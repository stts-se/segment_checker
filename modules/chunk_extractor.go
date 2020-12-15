package modules

import (
	"github.com/stts-se/segment_checker/modules/ffmpeg"
	"github.com/stts-se/segment_checker/modules/sox"
	"github.com/stts-se/segment_checker/protocol"
)

// ChunkExtractor extracts time chunks from an audio file, creating a subset of "phrases" from the file.
// For initialization, use NewChunkExtractor().
type ChunkExtractor interface {
	ProcessURLWithContext(payload protocol.SplitRequestPayload, encoding string) (protocol.AudioChunk, error)
}

func NewFfmpegChunkExtractor() (ChunkExtractor, error) {
	return ffmpeg.NewChunkExtractor()
}

func NewSoxChunkExtractor() (ChunkExtractor, error) {
	return sox.NewChunkExtractor()
}
