package protocol

import "encoding/json"

type SourcePayload struct {
	URL         string  `json:"url"`
	SegmentType string  `json:"segment_type"`
	Chunks      []Chunk `json:"chunks"`
}

type SegmentPayload struct {
	UUID        string `json:"uuid"`
	URL         string `json:"url"`
	SegmentType string `json:"segment_type"`
	Chunk       Chunk  `json:"chunk"`
}

type SplitRequestPayload struct {
	URL         string `json:"url"`
	SegmentType string `json:"segment_type"`
	// LeftContext in milliseconds
	LeftContext int64 `json:"left_context"`
	// RightContext in milliseconds
	RightContext int64 `json:"right_context"`
	Chunk        Chunk `json:"chunk"`
}

type Chunk struct {
	// Start time in milliseconds
	Start int64 `json:"start"`
	// End time in milliseconds
	End int64 `json:"end"`
}

type AudioChunk struct {
	AnnotationPayload
	// Audio is a base64 string representation of the audio
	Audio    string `json:"audio,omitempty"`
	FileType string `json:"file_type"`
	Offset   int64  `json:"offset"`
}

func (ac AudioChunk) PrettyMarshal() ([]byte, error) {
	copy := ac
	copy.Audio = ""
	return json.Marshal(copy)
}

// Annotation

type Status struct {
	Name      string `json:"name,attr"`
	Source    string `json:"source,attr"`
	Timestamp string `json:"timestamp,attr"`
}

type AnnotationPayload struct {
	SegmentPayload
	Labels        []string `json:"labels,omitempty"`
	CurrentStatus Status   `json:"current_status,omitempty"`
	StatusHistory []Status `json:"status_history,omitempty"`
	Comment       string   `json:"comment,omitempty"`
	Index         int64    `json:"index,omit"`
}

func (ap *AnnotationPayload) SetCurrentStatus(s Status) {
	if ap.CurrentStatus.Name != "" || ap.CurrentStatus.Source != "" {
		ap.StatusHistory = append(ap.StatusHistory, ap.CurrentStatus)
	}
	ap.CurrentStatus = s
}

// QueryPayload holds criteria used to search in the database
type QueryPayload struct {
	UserName      string   `json:"user_name"`
	RequestStatus []string `json:"request_status"`
	StepSize      int64    `json:"step_size"`
	CurrID        string   `json:"curr_id"`
	Context       int64    `json:"context,omitempty"`
}
