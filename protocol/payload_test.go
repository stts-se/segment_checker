package protocol

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSourceChunksPayload(t *testing.T) {
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
