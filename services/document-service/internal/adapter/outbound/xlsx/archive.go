package xlsx

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	zipDataDescriptorFlag = uint16(0x8)
	zip64ExtraID          = uint16(0x0001)
)

// part is one zip entry of the uploaded document. The original compressed bytes
// are kept so entries the domain never touches are copied into the saved
// archive with their content, compression method and metadata untouched.
type part struct {
	header  zip.FileHeader
	raw     []byte
	plain   []byte
	updated []byte
	removed bool
}

type archive struct {
	parts []*part
	index map[string]*part
}

func readArchive(content []byte) (*archive, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}
	result := &archive{index: make(map[string]*part, len(reader.File))}
	for _, file := range reader.File {
		raw, err := readRaw(file)
		if err != nil {
			return nil, fmt.Errorf("read entry %s: %w", file.Name, err)
		}
		entry := &part{header: file.FileHeader, raw: raw}
		result.parts = append(result.parts, entry)
		if _, exists := result.index[file.Name]; !exists {
			result.index[file.Name] = entry
		}
	}
	return result, nil
}

func (a *archive) get(name string) (*part, bool) {
	entry, ok := a.index[name]
	if !ok || entry.removed {
		return nil, false
	}
	return entry, true
}

func (a *archive) remove(name string) {
	if entry, ok := a.index[name]; ok {
		entry.removed = true
	}
}

func (a *archive) write() ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range a.parts {
		if entry.removed {
			continue
		}
		if err := entry.writeTo(writer); err != nil {
			return nil, fmt.Errorf("write entry %s: %w", entry.header.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (p *part) bytes() ([]byte, error) {
	if p.updated != nil {
		return p.updated, nil
	}
	if p.plain != nil {
		return p.plain, nil
	}
	decoded, err := decompress(p.header.Method, p.raw)
	if err != nil {
		return nil, fmt.Errorf("decompress entry %s: %w", p.header.Name, err)
	}
	p.plain = decoded
	return decoded, nil
}

func (p *part) replace(data []byte) {
	p.updated = data
}

func (p *part) writeTo(writer *zip.Writer) error {
	if p.updated == nil {
		header := p.header
		// archive/zip writes the sizes carried by the header, so the entry must
		// not advertise a trailing data descriptor, and its own zip64 field
		// replaces the original one.
		header.Flags &^= zipDataDescriptorFlag
		header.Extra = stripZip64Extra(header.Extra)
		entry, err := writer.CreateRaw(&header)
		if err != nil {
			return err
		}
		_, err = entry.Write(p.raw)
		return err
	}
	entry, err := writer.CreateHeader(&zip.FileHeader{
		Name:     p.header.Name,
		Method:   zip.Deflate,
		Modified: p.header.Modified,
	})
	if err != nil {
		return err
	}
	_, err = entry.Write(p.updated)
	return err
}

func readRaw(file *zip.File) ([]byte, error) {
	reader, err := file.OpenRaw()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func decompress(method uint16, raw []byte) ([]byte, error) {
	switch method {
	case zip.Store:
		return raw, nil
	case zip.Deflate:
		reader := flate.NewReader(bytes.NewReader(raw))
		defer func() { _ = reader.Close() }()
		return io.ReadAll(reader)
	default:
		return nil, fmt.Errorf("unsupported compression method %d", method)
	}
}

func stripZip64Extra(extra []byte) []byte {
	kept := make([]byte, 0, len(extra))
	for len(extra) >= 4 {
		id := binary.LittleEndian.Uint16(extra[0:2])
		size := int(binary.LittleEndian.Uint16(extra[2:4]))
		if len(extra) < 4+size {
			break
		}
		if id != zip64ExtraID {
			kept = append(kept, extra[:4+size]...)
		}
		extra = extra[4+size:]
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}
