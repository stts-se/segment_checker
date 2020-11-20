package modules

import (
	"os"
	"path"
	"testing"

	"github.com/stts-se/TillStud/segment_checker/protocol"
)

func TestChunk2File1MP3(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.mp3")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1587},
		{Start: 1587, End: 3885},
		{Start: 3885, End: 7647},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-1-mp3"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join("../test_data", tmpBase, "three_sentences_chunk0001.mp3"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0002.mp3"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0003.mp3"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}

func TestChunk2File1Wav(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "three_sentences.wav")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1600},
		{Start: 1600, End: 3922},
		{Start: 3922, End: 7684},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-1-wav"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join("../test_data", tmpBase, "three_sentences_chunk0001.wav"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0002.wav"),
		path.Join("../test_data", tmpBase, "three_sentences_chunk0003.wav"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}

func TestChunk2File_SubfolderMP3(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "subfolder_test/four_sentences.mp3")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1926},
		{Start: 1926, End: 4147},
		{Start: 4147, End: 6602},
		{Start: 6602, End: 9241},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-subfolder-mp3"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join(outDir, tmpBase, "four_sentences_chunk0001.mp3"),
		path.Join(outDir, tmpBase, "four_sentences_chunk0002.mp3"),
		path.Join(outDir, tmpBase, "four_sentences_chunk0003.mp3"),
		path.Join(outDir, tmpBase, "four_sentences_chunk0004.mp3"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}

func TestChunk2File_SubfolderWav(t *testing.T) {
	chunker, err := NewChunk2File()
	if err != nil {
		t.Errorf("got error from NewChunk2File: %v", err)
		return
	}
	fName := path.Join("../test_data", "subfolder_test/four_sentences.wav")
	chunks := []protocol.TimeChunk{
		{Start: 0, End: 1926},
		{Start: 1926, End: 4147},
		{Start: 4147, End: 6602},
		{Start: 6602, End: 9247},
	}

	outDir, _ := path.Split(fName)
	tmpBase := "tmp-chunkextractor-test-subfolder-wav"
	tmpDir := path.Join(outDir, tmpBase)
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll(tmpDir)

	got, err := chunker.Process(fName, chunks, tmpBase, "")
	if err != nil {
		t.Errorf("got error from Chunk2File.Process: %v", err)
		return
	}
	exp := []string{
		path.Join("../test_data", "subfolder_test", tmpBase, "four_sentences_chunk0001.wav"),
		path.Join("../test_data", "subfolder_test", tmpBase, "four_sentences_chunk0002.wav"),
		path.Join("../test_data", "subfolder_test", tmpBase, "four_sentences_chunk0003.wav"),
		path.Join("../test_data", "subfolder_test", tmpBase, "four_sentences_chunk0004.wav"),
	}
	if len(got) != len(exp) {
		t.Errorf("expected %v, got %v", exp, got)
		return
	}
	for i, exp0 := range exp {
		got0 := got[i]
		if got0 != exp0 {
			t.Errorf("expected %v, got %v", exp0, got0)
		}

	}
}
