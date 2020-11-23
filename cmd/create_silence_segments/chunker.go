package main

import (
	"fmt"
	//"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stts-se/segment_checker/protocol"
)

type Chunker struct {
}

var (
	silenceStartRE = regexp.MustCompile(".*] silence_start: ([0-9.]+) *")
	silenceEndRE   = regexp.MustCompile(".*] silence_end: ([0-9.]+) *")
	durationRE     = regexp.MustCompile("Duration: ([0-9]+):([0-9]{2}):([0-9]{2}[.][0-9]+)")
)

const extendChunk = 0 // extend all chunks by N ms before and after (N*2 ms in total)

// Process the audioFile into time chunks
func (ch Chunker) Process(audioFile string) ([]protocol.Chunk, error) {
	res := []protocol.Chunk{}

	//ffmpeg -i <LJUDFIL> -af silencedetect=noise=-50dB:d=1 -f null -
	cmd := exec.Command("ffmpeg", "-i", audioFile, "-af", "silencedetect=noise=-50dB:d=1", "-f", "null", "-")
	//log.Printf("chunker cmd: %v", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return res, fmt.Errorf("command %s failed : %#v", cmd, err)
	}

	var totalDuration int64

	currInterval := protocol.Chunk{Start: 0}
	for _, l := range strings.Split(string(out), "\n") {
		durM := durationRE.FindStringSubmatch(l)
		if len(durM) > 0 {
			//log.Println("dur", durM[0])
			h := durM[1]
			m := durM[2]
			s := durM[3]
			secFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			ms := int64(secFloat * 1000)
			fmtS := fmt.Sprintf("%sh%sm0s", h, m)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			dur, err := time.ParseDuration(fmtS)
			totalDuration = dur.Milliseconds() + ms
			//log.Println("totalDuration", totalDuration)
		}

		startM := silenceStartRE.FindStringSubmatch(l)
		if len(startM) > 0 {
			//log.Println("start", startM[0])
			s := startM[1]
			timePointFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			timePoint := int64(timePointFloat * 1000)
			if timePoint < totalDuration+extendChunk {
				timePoint = timePoint + extendChunk
			}
			currInterval.Start = timePoint
		}
		endM := silenceEndRE.FindStringSubmatch(l)
		if len(endM) > 0 {
			//log.Println("end", endM[0])
			s := endM[1]
			timePointFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return res, fmt.Errorf("couldn't parse ffmpeg output line %s : %v", s, err)
			}
			timePoint := int64(timePointFloat * 1000)
			if timePoint > extendChunk {
				timePoint = timePoint - extendChunk
			}
			currInterval.End = timePoint
			if currInterval.Start != 0 || timePoint != 0 {
				res = append(res, currInterval)
				currInterval = protocol.Chunk{}
			}
		}
	}
	if currInterval.End == 0 && currInterval.Start != 0 {
		currInterval.End = totalDuration
		res = append(res, currInterval)
	}
	return res, nil
}
