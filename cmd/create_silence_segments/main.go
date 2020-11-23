package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"

	"github.com/stts-se/segment_checker/protocol"
)

func main() {
	urlPrefix := "http://localhost:7371/audio/rispik/"
	outDir := "data/source"
	chunker := Chunker{}

	for _, fName := range os.Args[1:] {
		chunks, err := chunker.Process(fName)
		if err != nil {
			log.Fatalf("Got error from chunker.Process: %v", err)
		}
		source := protocol.SourcePayload{
			URL:         path.Join(urlPrefix, path.Base(fName)),
			SegmentType: "silence",
			Chunks:      chunks,
		}
		for i, chunk := range source.Chunks {
			id, err := uuid.NewUUID()
			if err != nil {
				log.Fatalf("Couldn't create uuid: %v", err)
			}
			segment := protocol.SegmentPayload{
				UUID:        fmt.Sprintf("%v", id),
				URL:         source.URL,
				SegmentType: source.SegmentType,
				Chunk:       chunk,
			}
			baseName := path.Base(strings.TrimSuffix(fName, path.Ext(fName)))
			outFile := path.Join(outDir, fmt.Sprintf("%s_%.5d.json", baseName, i))

			json, err := json.MarshalIndent(segment, " ", " ")
			if err != nil {
				log.Fatalf("Marshal failed: %v", err)
			}

			file, err := os.Create(outFile)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			file.Write(json)
			fmt.Fprintf(os.Stderr, "%s\n", outFile)
		}
	}
}
