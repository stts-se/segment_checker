package protocol

import (
	"encoding/json"
	"fmt"
	"reflect"
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
		Labels:        []string{"Bad sample"},
		CurrentStatus: Status{Source: "ringo", Name: "ok", Timestamp: "2020-12-01 14:34:37"},
		StatusHistory: []Status{
			{Source: "curt", Name: "skip", Timestamp: "2020-11-23 10:33:06"},
			{Source: "smirnoff", Name: "skip", Timestamp: "2020-11-30 17:41:15"},
		},
		Comment: "Konstigt ljud",
	}

	bts, err := json.MarshalIndent(payload, " ", " ")
	if err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	fmt.Println(string(bts))

	// change current status
	expectStatusHistory := payload.StatusHistory
	expectStatusHistory = append(expectStatusHistory, payload.CurrentStatus)
	newStatus := Status{Source: "pöbeln", Name: "ok", Timestamp: "2020-12-01 14:57:43"}
	payload.SetCurrentStatus(newStatus)

	bts, err = json.MarshalIndent(payload, " ", " ")
	if err != nil {
		t.Errorf("Marshal failed: %v", err)
	}
	fmt.Println(string(bts))

	if payload.CurrentStatus != newStatus {
		t.Errorf("Expected %v, found %v", newStatus, payload.CurrentStatus)
	}
	if !reflect.DeepEqual(expectStatusHistory, payload.StatusHistory) {
		t.Errorf("Expected %#v, found %#v", expectStatusHistory, payload.StatusHistory)
	}

}
