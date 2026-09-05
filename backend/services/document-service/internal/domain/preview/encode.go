package preview

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const gzipContentType = "application/gzip"

// Object is one sidecar file stored next to the generated workbook.
type Object struct {
	Name        string
	ContentType string
	Content     []byte
}

var errUnknownSheet = fmt.Errorf("preview sheet was not found")

// MetaKey is the object-store path of the gzipped sheet index.
func MetaKey(jobID string, fileID string) string {
	return fmt.Sprintf("jobs/%s/preview/%s/meta.json.gz", jobID, fileID)
}

// ChunkKey is the object-store path of one gzipped row window.
func ChunkKey(jobID string, fileID string, sheetIndex int, chunkIndex int) string {
	return fmt.Sprintf("jobs/%s/preview/%s/s%d/c%d.json.gz", jobID, fileID, sheetIndex, chunkIndex)
}

// RelativeChunkName is the name inside a decoded object list.
func RelativeChunkName(sheetIndex int, chunkIndex int) string {
	return fmt.Sprintf("s%d/c%d.json.gz", sheetIndex, chunkIndex)
}

// Encode compresses meta and every chunk so api-service never inflates a 100k
// sheet. Objects are ordered meta first, then chunks in sheet/chunk order.
func Encode(snapshot Snapshot) ([]Object, error) {
	objects := make([]Object, 0, 1+len(snapshot.Chunks)*4)
	metaBytes, err := gzipJSON(snapshot.Meta)
	if err != nil {
		return nil, fmt.Errorf("encode preview meta: %w", err)
	}
	objects = append(objects, Object{Name: "meta.json.gz", ContentType: gzipContentType, Content: metaBytes})
	for sheetIndex, chunks := range snapshot.Chunks {
		for chunkIndex, chunk := range chunks {
			payload, err := gzipJSON(chunk)
			if err != nil {
				return nil, fmt.Errorf("encode preview chunk s%d/c%d: %w", sheetIndex, chunkIndex, err)
			}
			objects = append(objects, Object{
				Name:        RelativeChunkName(sheetIndex, chunkIndex),
				ContentType: gzipContentType,
				Content:     payload,
			})
		}
	}
	return objects, nil
}

// Decode rebuilds a snapshot from Encode's object list. Used by tests and by
// the worker when it needs to assert a round-trip; api-service decodes one
// object at a time.
func Decode(objects []Object) (Snapshot, error) {
	var snapshot Snapshot
	chunksBySheet := map[int]map[int]Chunk{}
	for _, object := range objects {
		switch {
		case object.Name == "meta.json.gz":
			if err := gunzipJSON(object.Content, &snapshot.Meta); err != nil {
				return Snapshot{}, fmt.Errorf("decode preview meta: %w", err)
			}
		case strings.HasPrefix(object.Name, "s") && strings.HasSuffix(object.Name, ".json.gz"):
			sheetIndex, chunkIndex, err := parseChunkName(object.Name)
			if err != nil {
				return Snapshot{}, err
			}
			var chunk Chunk
			if err := gunzipJSON(object.Content, &chunk); err != nil {
				return Snapshot{}, fmt.Errorf("decode preview chunk %s: %w", object.Name, err)
			}
			if chunksBySheet[sheetIndex] == nil {
				chunksBySheet[sheetIndex] = map[int]Chunk{}
			}
			chunksBySheet[sheetIndex][chunkIndex] = chunk
		}
	}
	if len(snapshot.Meta.Sheets) == 0 && len(chunksBySheet) == 0 {
		return Snapshot{}, fmt.Errorf("preview objects contained no meta")
	}
	snapshot.Chunks = make([][]Chunk, len(snapshot.Meta.Sheets))
	for sheetIndex := range snapshot.Meta.Sheets {
		byIndex := chunksBySheet[sheetIndex]
		if len(byIndex) == 0 {
			continue
		}
		maxIndex := -1
		for index := range byIndex {
			if index > maxIndex {
				maxIndex = index
			}
		}
		list := make([]Chunk, maxIndex+1)
		for index, chunk := range byIndex {
			list[index] = chunk
		}
		snapshot.Chunks[sheetIndex] = list
	}
	return snapshot, nil
}

func parseChunkName(name string) (int, int, error) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(name, "s"), ".json.gz")
	parts := strings.Split(trimmed, "/c")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("preview chunk name %q", name)
	}
	sheetIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("preview chunk name %q", name)
	}
	chunkIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("preview chunk name %q", name)
	}
	return sheetIndex, chunkIndex, nil
}

func gzipJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func gunzipJSON(raw []byte, dest any) error {
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, dest)
}
