package sox

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/stts-se/segment_checker/protocol"
)

// ChunkExtractor extracts time chunks from an audio file, creating a subset of "phrases" from the file.
// For initialization, use NewChunkExtractor().
type ChunkExtractor struct {
}

// NewChunkExtractor creates a new ChunkExtractor after first checking that the sox command exists
func NewChunkExtractor() (ChunkExtractor, error) {
	if err := soxEnabled(); err != nil {
		return ChunkExtractor{}, err
	}
	return ChunkExtractor{}, nil
}

func (ch ChunkExtractor) processWithContext(processFunc func(string, []protocol.Chunk, string) ([][]byte, error), audioFile string, chunk protocol.Chunk, leftContext, rightContext int64, encoding string) (protocol.AudioChunk, error) {
	offset := chunk.Start - leftContext
	if offset < 0 {
		offset = 0
	}
	processChunk := protocol.Chunk{
		Start: offset,
		End:   chunk.End + rightContext,
	}

	ext := filepath.Ext(audioFile)
	ext = strings.TrimPrefix(ext, ".")
	if encoding == "" {
		encoding = ext
	}
	btss, err := processFunc(audioFile, []protocol.Chunk{processChunk}, encoding)
	if err != nil {
		return protocol.AudioChunk{}, err
	}

	if len(btss) != 1 {
		return protocol.AudioChunk{}, fmt.Errorf("expected one byte array, found %d", len(btss))
	}

	bts := btss[0]

	//ioutil.WriteFile("chunk_extractor_debug.wav", bts, 0644)

	res := protocol.AudioChunk{
		Audio:    base64.StdEncoding.EncodeToString(bts),
		FileType: encoding,
		Offset:   offset,
	}
	res.Chunk = protocol.Chunk{
		Start: chunk.Start - offset,
		End:   chunk.End - offset,
	}
	return res, nil

}

// ProcessFileWithContext an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFileWithContext(audioFile string, chunk protocol.Chunk, leftContext, rightContext int64, encoding string) (protocol.AudioChunk, error) {
	return ch.processWithContext(ch.ProcessFile, audioFile, chunk, leftContext, rightContext, encoding)
}

// ProcessURLWithContext an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURLWithContext(payload protocol.SplitRequestPayload, encoding string) (protocol.AudioChunk, error) {
	return ch.processWithContext(ch.ProcessURL, payload.URL, payload.Chunk, payload.LeftContext, payload.RightContext, encoding)
}

// ProcessURL an audioURL, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessURL(audioURL string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
	res := [][]byte{}
	for _, chunk := range chunks {

		ext := filepath.Ext(audioURL)
		ext = strings.TrimPrefix(ext, ".")
		if encoding == "" {
			encoding = ext
		}

		startFloat := float64(chunk.Start) / 1000.0
		endFloat := float64(chunk.End) / 1000.0
		duration := endFloat - startFloat
		fmt.Println("ba fatt", startFloat, endFloat, duration)
		urlResp, err := http.Get(audioURL)
		if err != nil {
			return res, fmt.Errorf("audio URL %s not reachable : %v", audioURL, err)
		}
		defer urlResp.Body.Close()
		if urlResp.StatusCode != http.StatusOK {
			return res, fmt.Errorf("audio URL %s not reachable (status %s)", audioURL, urlResp.Status)
		}

		//cat <infile> | sox - -t flac - trim <start> <duration>
		args := []string{"-", "-t", encoding, "-", "trim", fmt.Sprintf("%v", startFloat), fmt.Sprintf("%v", duration)}
		proc := exec.Command(SoxCmd, args...)
		//proc := exec.Command("cat")
		fmt.Println("proc:", proc)
		stdin, err := proc.StdinPipe()
		if err != nil {
			return res, fmt.Errorf("couldn't get stdin from process : %v", err)
		}
		bts, err := ioutil.ReadAll(urlResp.Body)
		if err != nil {
			return res, fmt.Errorf("couldn't get read response body : %v", err)
		}
		fmt.Println("ba fatt", len(bts))
		var goFuncErr error
		go func() {
			defer stdin.Close()
			w := bufio.NewWriter(stdin)
			_, err = w.Write(bts)
			//_, err = w.Write([]byte("hej"))
			if err != nil {
				fmt.Println("????", err)
				goFuncErr = fmt.Errorf("couldn't get write to process stdin : %v", err)
			}
			w.Flush()
		}()
		proc.Wait()
		proc.Stderr = os.Stderr
		out, err := proc.Output()
		if err != nil {
			return res, fmt.Errorf("command %s failed : %#v", proc, err)
		}
		if goFuncErr != nil {
			return res, goFuncErr
		}
		res = append(res, out)
	}
	return res, nil
}

// ProcessFile an audioFile, extracting the specified chunks to slices of byte
func (ch ChunkExtractor) ProcessFile(audioFile string, chunks []protocol.Chunk, encoding string) ([][]byte, error) {
	res := [][]byte{}
	for _, chunk := range chunks {

		ext := filepath.Ext(audioFile)
		ext = strings.TrimPrefix(ext, ".")
		if encoding == "" {
			encoding = ext
		}

		startFloat := float64(chunk.Start) / 1000.0
		endFloat := float64(chunk.End) / 1000.0
		duration := endFloat - startFloat
		//sox <infile> -t flac - trim <start> <duration>
		args := []string{audioFile, "-t", encoding, "-", "trim", fmt.Sprintf("%v", startFloat), fmt.Sprintf("%v", duration)}
		cmd := exec.Command(SoxCmd, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return res, fmt.Errorf("command %s failed : %#v", cmd, err)
		}
		res = append(res, out)

	}
	return res, nil
}
