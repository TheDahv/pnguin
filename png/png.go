// Package png implements a parser for PNG files and operations on those files
// according to https://en.wikipedia.org/wiki/Portable_Network_Graphics
package png

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
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
	ctHdr  = []byte{'I', 'H', 'D', 'R'}
	ctPlte = []byte{'P', 'L', 'T', 'E'}
	ctDat  = []byte{'I', 'D', 'A', 'T'}
	ctEnd  = []byte{'I', 'E', 'N', 'D'}
	ctBkgd = []byte{'b', 'K', 'G', 'D'}
	ctChrm = []byte{'c', 'H', 'R', 'M'}
	ctDSig = []byte{'d', 'S', 'I', 'G'}
	ctExif = []byte{'e', 'X', 'I', 'f'}
	ctGama = []byte{'g', 'A', 'M', 'A'}
	ctHist = []byte{'h', 'I', 'S', 'T'}
	ctIccp = []byte{'i', 'C', 'C', 'P'}
	ctItxt = []byte{'i', 'T', 'X', 't'}
	ctPhys = []byte{'p', 'H', 'Y', 's'}
	ctSbit = []byte{'s', 'B', 'I', 'T'}
	ctSplt = []byte{'s', 'P', 'L', 'T'}
	ctSrgb = []byte{'s', 'R', 'G', 'B'}
	ctSter = []byte{'s', 'T', 'E', 'R'}
	ctText = []byte{'t', 'E', 'X', 't'}
	ctTime = []byte{'t', 'I', 'M', 'E'}
	ctTrns = []byte{'t', 'R', 'N', 'S'}
	ctZtxt = []byte{'z', 'T', 'X', 't'}
)

type chunkType uint32

// Types of chunks that comprise the image and convey information about the
// content.
// https://en.wikipedia.org/wiki/Portable_Network_Graphics#.22Chunks.22_within_the_file
const (
	ChunkTypeUnknown chunkType = iota

	// Critical types
	ChunkTypeHeader
	ChunkTypePalette
	ChunkTypeData
	ChunkTypeEnd

	// Ancillary types
	ChunkTypeBkgdColor
	ChunkTypeChromaticity
	ChunkTypeDigiSignal
	ChunkTypeExif
	ChunkTypeGamma
	ChunkTypeHistogram
	ChunkTypeICC
	ChunkTypeTxtUTF8
	ChunkTypePxSize
	ChunkTypeSigBits
	ChunkTypeSugPalette
	ChunkTypeRGB
	ChunkTypeStereo
	ChunkTypeTxtISO8859
	ChunkTypeTimeChanged
	ChunkTypeTransparency
	ChunkTypeTxtCompressed
)

// String converts chunk types to a human-friendly representation
func (ct chunkType) String() string {
	switch ct {
	case ChunkTypeHeader:
		return "IHDR (Header)"
	case ChunkTypePalette:
		return "PLTE (Pallette)"
	case ChunkTypeData:
		return "IDAT (Data)"
	case ChunkTypeEnd:
		return "IEND (Image End)"
	case ChunkTypeBkgdColor:
		return "bKGD (Default Background Color)"
	case ChunkTypeChromaticity:
		return "cHRM (Chromaticity)"
	case ChunkTypeDigiSignal:
		return "dSIG (Digital Signatures)"
	case ChunkTypeExif:
		return "eXIf (Exif)"
	case ChunkTypeGamma:
		return "gAMA (Gamma)"
	case ChunkTypeHistogram:
		return "hIST (Color Histogram)"
	case ChunkTypeICC:
		return "iCCP (ICC Color Profile)"
	case ChunkTypeTxtUTF8:
		return "iTXt (UTF-8 Keyword Text)"
	case ChunkTypePxSize:
		return "pHYs (Intended Pixel Size)"
	case ChunkTypeSigBits:
		return "sBIT (Color-Accuracy)"
	case ChunkTypeSugPalette:
		return "sPLT (Suggested Palette)"
	case ChunkTypeRGB:
		return "sRGB (sRGB Color Space)"
	case ChunkTypeStereo:
		return "sTER (Stereo-Image Indicator)"
	case ChunkTypeTxtISO8859:
		return "tEXt (ISO/IEC 885901 Text)"
	case ChunkTypeTimeChanged:
		return "tIME (Last Changed Time)"
	case ChunkTypeTransparency:
		return "tRNS (Transparency)"
	case ChunkTypeTxtCompressed:
		return "zTXt (Compressed Text)"
	default:
		return "Unknown"
	}
}

// Parser knows how to parse and operate on PNG files
type Parser struct {
	Path string
	rc   io.ReadCloser
	br   *bufio.Reader
	data []Chunk
}

// Chunk holds information and data in an image.
type Chunk struct {
	Length [4]byte
	CRC    [4]byte
	Type   chunkType
	Data   []byte
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

// WalkChunks iterates over the parsed chunks in the file. Each is handed to the
// iteratee function, which can return true or false to indicate whether
// iteration should continue.
func (p *Parser) WalkChunks(fn func(ch Chunk) bool) {
	for _, ch := range p.data {
		if cont := fn(ch); !cont {
			break
		}
	}
}

// Close closes the internal file
func (p *Parser) Close() error {
	return p.rc.Close()
}

// PrintHeader outputs header chunks to stdout
func (p *Parser) PrintHeader() {
	for _, ch := range p.data {
		if ch.Type == ChunkTypeHeader {
			fmt.Printf("%s Header\n", p.Path)
			hdr, _ := parseHeader(ch.Data) // TODO handle
			fmt.Fprintf(os.Stdout, "Width\t%d\n", hdr.Width)
			fmt.Fprintf(os.Stdout, "Height\t%d\n", hdr.Height)
			fmt.Fprintf(os.Stdout, "Bit Depth\t%d\n", hdr.BitDepth)
			fmt.Fprintf(os.Stdout, "Color Type\t%d\n", hdr.ColorType)
			fmt.Fprintf(os.Stdout, "Compression Method\t%d\n", hdr.CompressionMethod)
			fmt.Fprintf(os.Stdout, "Filter Method\t%d\n", hdr.BitDepth)
			fmt.Fprintf(os.Stdout, "Interlace Method\t%d\n", hdr.InterlaceMethod)
			fmt.Fprintln(os.Stdout)
		}
	}
}

// StripTags returns a version of the input file with all non-critical chunks
// and metadata removed.
func (p *Parser) StripTags() io.Reader {
	r, w := io.Pipe()

	go func() {
		if _, err := w.Write(pngHeader); err != nil {
			w.CloseWithError(fmt.Errorf("unable to write PNG header: %v", err))
			return
		}

		var passThrough = map[chunkType]bool{
			ChunkTypeHeader:  true,
			ChunkTypePalette: true,
			ChunkTypeData:    true,
			ChunkTypeEnd:     true,
		}

		var err error
		p.WalkChunks(func(ch Chunk) bool {
			if _, ok := passThrough[ch.Type]; ok {
				var typeBytes []byte
				switch ch.Type {
				case ChunkTypeHeader:
					typeBytes = ctHdr
				case ChunkTypePalette:
					typeBytes = ctPlte
				case ChunkTypeData:
					typeBytes = ctDat
				case ChunkTypeEnd:
					typeBytes = ctEnd
				}
				if _, e := w.Write(ch.Length[:]); e != nil {
					err = fmt.Errorf("unable to write chunk length: %v", e)
					return false
				}
				if _, e := w.Write(typeBytes); e != nil {
					err = fmt.Errorf("unable to write chunk type: %v", e)
					return false
				}
				if _, e := w.Write(ch.Data[:]); e != nil {
					err = fmt.Errorf("unable to write chunk data: %v", e)
					return false
				}
				if _, e := w.Write(ch.CRC[:]); e != nil {
					err = fmt.Errorf("unable to write chunk CRC: %v", e)
					return false
				}
			}
			return true
		})

		w.CloseWithError(err)
	}()

	return r
}

// Chunks returns a slice of chunks parsed from the PNG
func (p *Parser) chunks() ([]Chunk, error) {
	var chunks []Chunk

	b, err := p.IsPNG()
	if err != nil {
		return chunks, err
	}
	if !b {
		return chunks, errors.New("input not a PNG")
	}

	fileHdr := make([]byte, 8)
	if c, err := io.ReadFull(p.br, fileHdr); err != nil || c != 8 {
		return chunks, fmt.Errorf("unable to read header: %v", err)
	}

	for {
		c := Chunk{}

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
		return ChunkTypeHeader
	}
	if bytes.Compare(ct[:], ctPlte) == 0 {
		return ChunkTypePalette
	}
	if bytes.Compare(ct[:], ctDat) == 0 {
		return ChunkTypeData
	}
	if bytes.Compare(ct[:], ctEnd) == 0 {
		return ChunkTypeEnd
	}
	if bytes.Compare(ct[:], ctBkgd) == 0 {
		return ChunkTypeBkgdColor
	}
	if bytes.Compare(ct[:], ctChrm) == 0 {
		return ChunkTypeChromaticity
	}
	if bytes.Compare(ct[:], ctDSig) == 0 {
		return ChunkTypeDigiSignal
	}
	if bytes.Compare(ct[:], ctExif) == 0 {
		return ChunkTypeExif
	}
	if bytes.Compare(ct[:], ctGama) == 0 {
		return ChunkTypeGamma
	}
	if bytes.Compare(ct[:], ctHist) == 0 {
		return ChunkTypeHistogram
	}
	if bytes.Compare(ct[:], ctIccp) == 0 {
		return ChunkTypeICC
	}
	if bytes.Compare(ct[:], ctItxt) == 0 {
		return ChunkTypeTxtUTF8
	}
	if bytes.Compare(ct[:], ctPhys) == 0 {
		return ChunkTypePxSize
	}
	if bytes.Compare(ct[:], ctSbit) == 0 {
		return ChunkTypeSigBits
	}
	if bytes.Compare(ct[:], ctSplt) == 0 {
		return ChunkTypeSugPalette
	}
	if bytes.Compare(ct[:], ctSrgb) == 0 {
		return ChunkTypeRGB
	}
	if bytes.Compare(ct[:], ctSter) == 0 {
		return ChunkTypeStereo
	}
	if bytes.Compare(ct[:], ctText) == 0 {
		return ChunkTypeTxtISO8859
	}
	if bytes.Compare(ct[:], ctTime) == 0 {
		return ChunkTypeTimeChanged
	}
	if bytes.Compare(ct[:], ctTrns) == 0 {
		return ChunkTypeTransparency
	}
	if bytes.Compare(ct[:], ctZtxt) == 0 {
		return ChunkTypeTxtCompressed
	}

	return ChunkTypeUnknown
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
