package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func FetchFromParser(demData []byte, parserURL string) ([]RawEvent, error) {
	resp, err := http.Post(parserURL, "application/octet-stream", bytes.NewReader(demData))
	if err != nil {
		return nil, fmt.Errorf("parser request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("parser returned %d: %s", resp.StatusCode, string(body))
	}

	return readNDJSON(resp.Body)
}

func ReadNDJSONFile(path string) ([]RawEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return readNDJSON(bytes.NewReader(data))
}

func readNDJSON(r io.Reader) ([]RawEvent, error) {
	var events []RawEvent
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		var e RawEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("parse line: %w", err)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ndjson: %w", err)
	}
	return events, nil
}
