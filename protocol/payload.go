package protocol

type SourcePayload struct {
	URL         string  `json:"url"`
	SegmentType string  `json:"segment_type"`
	Chunks      []Chunk `json:"chunks"`
}

type Chunk struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}
