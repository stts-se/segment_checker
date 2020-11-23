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
	Labels  []string `json:"labels"`
	Status  Status   `json:"status"`
	Comment string   `json:"comment"`
}
