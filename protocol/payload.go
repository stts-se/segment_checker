package protocol

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
	SegmentPayload
	Audio    string `json:"audio"`
	FileType string `json:"file_type"`
	Offset   int64  `json:"offset"`
}

// Annotation

type Status struct {
	Name      string `json:"name,attr"`
	Source    string `json:"source,attr"`
	Timestamp string `json:"timestamp,attr"`
}

type AnnotationPayload struct {
	SegmentPayload
	Labels        []string `json:"labels"`
	CurrentStatus Status   `json:"current_status"`
	StatusHistory []Status `json:"status_history"`
	Comment       string   `json:"comment"`
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
	Context       int64    `json:"context"`
}
