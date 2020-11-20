package modules

import (
	"path"
	"testing"

	"github.com/stts-se/TillStud/segment_checker/protocol"
)

func TestChunkExtractorToMP3(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.Process(fName, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{13183, 18826, 30738} // approximate byte len
	if len(got) != len(expLens) {
		t.Errorf("expected %v, got %v", expLens, got)
		return
	}
	for i, exp0 := range expLens {
		got0 := got[i]
		max := exp0 + 200
		min := exp0 - 200
		if len(got0) < min || len(got0) > max {
			t.Errorf("expected a value between %v and %v, got %v", min, max, len(got0))
		}

	}
}

func TestChunkExtractorToWAV(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.Process(fName, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{140052, 202762, 331886} // approximate byte len
	if len(got) != len(expLens) {
		t.Errorf("expected %v, got %v", expLens, got)
		return
	}
	for i, exp0 := range expLens {
		got0 := got[i]
		max := exp0 + 200
		min := exp0 - 200
		if len(got0) < min || len(got0) > max {
			t.Errorf("expected a value between %v and %v, got %v", min, max, len(got0))
		}

	}
}
