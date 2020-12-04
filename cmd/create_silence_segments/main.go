package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/stts-se/segment_checker/protocol"
)

func main() {

	cmd := "create_silence_segments"

	project := flag.String("project", "rispik", "project name")
	target := flag.Int("target", 0, "Target size")
	//urlPrefixFlag := flag.String("urlprefix", "http://localhost:7371/audio/${project}", "URL prefix")
	urlPrefixFlag := flag.String("urlprefix", "http://localhost:7371/audio/rispik", "URL prefix")
	outDirFlag := flag.String("outdir", "data/${project}/source", "Output `directory`")

	help := flag.Bool("help", false, "Print usage and exit")
	h := flag.Bool("h", false, "Print usage and exit")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <options> <audio files>\n", cmd)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	if *help || *h {
		flag.Usage()
		os.Exit(0)
	}

	urlPrefix := strings.Replace(*urlPrefixFlag, "${project}", *project, 1)
	outDir := strings.Replace(*outDirFlag, "${project}", *project, 1)
	os.MkdirAll(outDir, os.ModePerm)
	fmt.Fprintf(os.Stderr, "Project: %s\n", *project)
	fmt.Fprintf(os.Stderr, "Target: %v\n", *target)
	fmt.Fprintf(os.Stderr, "URL prefix: %s\n", urlPrefix)
	fmt.Fprintf(os.Stderr, "Output directory: %s\n", outDir)
	fmt.Fprintf(os.Stderr, "\n")

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	chunker := Chunker{}

	counter := 0
	counterLock := sync.Mutex{}

	fmt.Fprintf(os.Stderr, "Creating files ")
	for counter < *target || *target == 0 {
		for _, fName := range flag.Args() {
			func() {
				chunks, err := chunker.Process(fName)
				if err != nil {
					log.Fatalf("Got error from chunker.Process: %v after %d created files", err, counter)
				}
				source := protocol.SourcePayload{
					URL:         path.Join(urlPrefix, path.Base(fName)),
					SegmentType: "silence",
					Chunks:      chunks,
				}
				for _, chunk := range source.Chunks {
					id, err := uuid.NewUUID()
					if err != nil {
						log.Fatalf("Couldn't create uuid: %v", err)
					}
					segment := protocol.SegmentPayload{
						ID:          fmt.Sprintf("%v", id),
						URL:         source.URL,
						SegmentType: source.SegmentType,
						Chunk:       chunk,
					}
					outFile := path.Join(outDir, fmt.Sprintf("%s.json", id))

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
					//fmt.Fprintf(os.Stderr, "%s\n", outFile)
					counterLock.Lock()
					counter++
					counterLock.Unlock()
					if counter%100 == 0 {
						fmt.Fprintf(os.Stderr, ".")
					}
				}
			}()
		}
		if *target == 0 {
			break
		}
	}
	fmt.Fprintf(os.Stderr, " done\nCreated %d files\n", counter)
}
