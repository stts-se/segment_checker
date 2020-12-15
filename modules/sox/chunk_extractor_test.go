package sox

import (
	"fmt"
	//"net/http"
	"path"
	"testing"
	//"time"

	"github.com/stts-se/segment_checker/protocol"
)

func TestChunkExtractorFileMP3(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessFile(fName, chunks, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	expLens := []int{12956, 18599, 30511} // approximate byte len
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

func TestChunkExtractorFileWAV(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessFile(fName, chunks, "")
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

func TestChunkExtractorURLWAV(t *testing.T) {
	//srv := "localhost:9876"
	srv := "localhost:7381"

	//go func() {
	// fmt.Println("SLEEP START")
	// time.Sleep(1000)
	// fmt.Println("SLEEP END")

	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	url := fmt.Sprintf("http://%s/three_sentences.wav", srv)
	chunks := []protocol.Chunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	got, err := chunker.ProcessURL(url, chunks, "")
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
	fmt.Println("TESTS DONE")
	//}()

	// http.Handle("/", http.FileServer(http.Dir("../test_data")))
	// fmt.Println("STARTING SRV")
	// if err := http.ListenAndServe(srv, nil); err != nil {
	// 	fmt.Errorf("http.ListenAndServe %v", err)
	// }

}

func TestChunkExtractorFileWithContext(t *testing.T) {
	chunker, err := NewChunkExtractor()
	if err != nil {
		t.Errorf("got error from NewChunkExtractor: %v", err)
		return
	}
	file := "../test_data/three_sentences.wav"
	leftContext := int64(100)
	rightContext := int64(100)
	var chunk protocol.Chunk
	var got protocol.AudioChunk

	//
	chunk = protocol.Chunk{Start: 165, End: 261}

	got, err = chunker.ProcessFileWithContext(file, chunk, leftContext, rightContext, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Chunk.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Chunk.Start)
	}
	if got.Chunk.End != 196 {
		t.Errorf("expected %v, got %v", 196, got.Chunk.End)
	}

	//
	chunk = protocol.Chunk{Start: 405, End: 514}

	got, err = chunker.ProcessFileWithContext(file, chunk, leftContext, rightContext, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Chunk.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Chunk.Start)
	}
	if got.Chunk.End != 209 {
		t.Errorf("expected %v, got %v", 209, got.Chunk.End)
	}

	//
	chunk = protocol.Chunk{Start: 405, End: 514}

	got, err = chunker.ProcessFileWithContext(file, chunk, leftContext, rightContext, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Chunk.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Chunk.Start)
	}
	if got.Chunk.End != 209 {
		t.Errorf("expected %v, got %v", 209, got.Chunk.End)
	}

	//
	chunk = protocol.Chunk{Start: 767, End: 826}

	got, err = chunker.ProcessFileWithContext(file, chunk, leftContext, rightContext, "")
	if err != nil {
		t.Errorf("got error from ChunkExtractor.Process: %v", err)
		return
	}
	if got.Chunk.Start != 100 {
		t.Errorf("expected %v, got %v", 100, got.Chunk.Start)
	}
	if got.Chunk.End != 159 {
		t.Errorf("expected %v, got %v", 159, got.Chunk.End)
	}
}
