package protocol

// type Message struct {
// 	Level string `json:"level,omitempty"`
// 	Text  string `json:"text,omitempty"`
// }

type SourcePayload struct {
	URL         string  `json:"url"`
	SegmentType string  `json:"segment_type"`
	Chunks      []Chunk `json:"chunks"`
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
	Audio    string `json:"audio"`
	FileType string `json:"file_type"`
	Chunk    Chunk  `json:"chunk"`
}
