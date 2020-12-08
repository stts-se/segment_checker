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

	"github.com/stts-se/segment_checker/protocol"
)

func createID(audioFile string, chunkIndex int) string {
	f := strings.Replace(path.Base(audioFile), ".", "_", -1)
	return fmt.Sprintf("%s_%04d", f, (chunkIndex + 1))
}

func main() {

	cmd := "create_silence_segments"

	project := flag.String("project", "", "Project name")
	target := flag.Int("target", 0, "Target size (can generate duplicated data for performance testing)")
	urlPrefixFlag := flag.String("urlprefix", "http://localhost:7381/", "URL prefix")
	outDirFlag := flag.String("outdir", "data/${project}/source", "Output `directory`")

	help := flag.Bool("help", false, "Print usage and exit")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <options> <audio files>\n", cmd)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *project == "" {
		fmt.Fprintf(os.Stderr, "Required flag project is not set: project\n")
		flag.Usage()
		os.Exit(1)
	}

	urlPrefix := strings.Replace(*urlPrefixFlag, "${project}", *project, 1)
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix = urlPrefix + "/"
	}
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
					URL:         fmt.Sprintf("%s%s", urlPrefix, path.Base(fName)),
					SegmentType: "silence",
					Chunks:      chunks,
				}
				for i, chunk := range source.Chunks {
					id := createID(fName, i)
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
