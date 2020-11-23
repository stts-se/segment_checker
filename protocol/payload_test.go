package protocol

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSourcePayload(t *testing.T) {
	payload := SourcePayload{
		URL:         "http://localhost/audio/fgfgfgfgf.wav",
		SegmentType: "silence",
		Chunks: []Chunk{
			{Start: 301, End: 351},
			{Start: 1908, End: 1958},
		},
	}

	bts, err := json.MarshalIndent(payload, " ", " ")
	if err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	fmt.Println(string(bts))

}

func TestAnnotationPayload(t *testing.T) {
	payload := AnnotationPayload{
		SegmentPayload: SegmentPayload{
			URL:         "http://localhost/audio/fgfgfgfgf.wav",
			SegmentType: "silence",
			Chunk:       Chunk{Start: 301, End: 351},
		},
		Labels:  []string{"Bad sample"},
		Status:  Status{Source: "curt", Name: "skip", Timestamp: "2020-11-23 10:33:06"},
		Comment: "Konstigt ljud",
	}

	bts, err := json.MarshalIndent(payload, " ", " ")
	if err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	fmt.Println(string(bts))

}
