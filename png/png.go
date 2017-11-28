// package png implements a parser for PNG files and operations on those files
// according to https://en.wikipedia.org/wiki/Portable_Network_Graphics
package png

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	pngHeader = []byte{
		'\x89',
		'\x50',
		'\x4e',
		'\x47',
		'\x0d',
		'\x0a',
		'\x1a',
		'\x0a',
	}
	ctHdr = []byte{'I', 'H', 'D', 'R'}
)

type chunkType int

const (
	chunkTypeUnknown chunkType = iota

	// Critical types
	chunkTypeHeader
	chunkTypePalette
	chunkTypeData
	chunkTypeEnd

	// Ancillary types
	chunkTypeBkgdColor
	chunkTypeChromaticity
	chunkTypeDigiSignal
	chunkTypeExif
	chunkTypeGamma
	chunkTypeHistogram
	chunkTypeICC
	chunkTypeTxtUTF8
	chunkTypePxSize
	chunkTypeSigBits
	chunkTypeSugPalette
	chunkTypeRGB
	chunkTypeStereo
	chunkTypeTxtISO8859
	chunkTypeTimeChanged
	chunkTypeTransparency
	chunkTypeTxtCompressed
)

// Parser knows how to parse and operate on PNG files
type Parser struct {
	Path string
	rc   io.ReadCloser
	br   *bufio.Reader
	data []chunk
}

// TODO look at field byte padding for this
type chunk struct {
	Length []byte
	CRC    []byte
	Data   []byte
	Type   chunkType
}

// it contains (in this order) the image's width, height, bit depth, color type,
// compression method, filter method, and interlace method (13 data bytes total)
type headerChunk struct {
	Width             uint32
	Height            uint32
	BitDepth          byte
	ColorType         byte
	CompressionMethod byte
	FilterMethod      byte
	InterlaceMethod   byte
}

// New returns a new parser on the given input
func New(path string, rc io.ReadCloser) *Parser {
	return &Parser{
		Path: path,
		rc:   rc,
		br:   bufio.NewReader(rc),
	}
}

// IsPNG checks for the required headers in the input. It does not advance the
// reader.
func (p *Parser) IsPNG() (bool, error) {
	b, err := p.br.Peek(8)

	if err != nil {
		return false, err
	}

	return bytes.Compare(pngHeader, b) == 0, nil
}

// Parse reads the chunks from the input
func (p *Parser) Parse() error {
	chunks, err := p.chunks()
	if err != nil {
		return err
	}

	p.data = chunks
	return nil
}

// Close closes the internal file
func (p *Parser) Close() error {
	return p.rc.Close()
}

// PrintHeader outputs header chunks to stdout
func (p *Parser) PrintHeader() {
	for _, ch := range p.data {
		if ch.Type == chunkTypeHeader {
			fmt.Printf("%s Header\n", p.Path)
			hdr, _ := parseHeader(ch.Data) // TODO handle
			fmt.Printf("Width\t%d\n", hdr.Width)
			fmt.Printf("Height\t%d\n", hdr.Height)
			fmt.Printf("Bit Depth\t%d\n", hdr.BitDepth)
			fmt.Printf("Color Type\t%d\n", hdr.ColorType)
			fmt.Printf("Compression Method\t%d\n", hdr.CompressionMethod)
			fmt.Printf("Filter Method\t%d\n", hdr.BitDepth)
			fmt.Printf("Interlace Method\t%d\n", hdr.InterlaceMethod)
			fmt.Println()
		}
	}
}

// Chunks returns a slice of chunks parsed from the PNG
func (p *Parser) chunks() ([]chunk, error) {
	var chunks []chunk

	b, err := p.IsPNG()
	if err != nil {
		return chunks, err
	}
	if !b {
		return chunks, errors.New("input not a PNG")
	}

	var fileHdr []byte
	for i := 0; i < 8; i++ {
		b, err := p.br.ReadByte()
		if err != nil {
			return chunks, fmt.Errorf("unable to read header: %v", err)
		}
		fileHdr = append(fileHdr, b)
	}

	for {
		c := chunk{
			Length: make([]byte, 4),
			CRC:    make([]byte, 4),
		}

		// Read LENGTH
		read, err := io.ReadFull(p.br, c.Length[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return chunks, fmt.Errorf("unable to read chunk length: %v", err)
		}

		// Read TYPE
		chType := make([]byte, 4)
		read, err = io.ReadFull(p.br, chType)
		if err == io.EOF {
			break
		}
		if err != nil {
			return chunks, fmt.Errorf("unable to read chunk type: %v", err)
		}
		c.Type = getChunkType(chType)

		// Read DATA
		l := binary.BigEndian.Uint32(c.Length[:])
		data := make([]byte, l)
		read, err = io.ReadFull(p.br, data)
		if err == io.EOF {
			break
		}
		if err != nil {
			return chunks, fmt.Errorf("unable to read chunk data: %v", err)
		}
		c.Data = data

		// Read CRC
		read, err = io.ReadFull(p.br, c.CRC[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return chunks, fmt.Errorf("unable to read chunk CRC: %v", err)
		}
		if l := len(c.Length); read != l {
			return chunks, fmt.Errorf(
				"short read on chunk CRC (got %d bytes, expected %d)", read, l)
		}

		chunks = append(chunks, c)
	}

	return chunks, nil
}

func getChunkType(ct []byte) chunkType {
	if bytes.Compare(ct[:], ctHdr) == 0 {
		return chunkTypeHeader
	}

	return chunkTypeUnknown
}

func parseHeader(data []byte) (headerChunk, error) {
	var hdr headerChunk
	if l := len(data); l != 13 {
		return hdr, fmt.Errorf("got %d bytes for header chunk, expected %d",
			l, 13)
	}

	hdr.Width = binary.BigEndian.Uint32(data[0:4])
	hdr.Height = binary.BigEndian.Uint32(data[4:8])
	hdr.BitDepth = data[8]
	hdr.ColorType = data[9]
	hdr.CompressionMethod = data[10]
	hdr.FilterMethod = data[11]
	hdr.InterlaceMethod = data[12]

	return hdr, nil
}
