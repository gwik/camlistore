package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	//"hash/crc32"
)

const (
	flagContinued = 0x01
	flagFirst     = 0x02
	flagLast      = 0x04
)

var capturePattern = []byte("OggS")

var codecTheora = []byte("\x80theora")

// pageHeader starts with OggS
type pageHeader struct {
	version  uint8      // stream_structure_version (4)
	flags    uint8      // header_type_flag (5)
	pos      uint64     // absolute granule position (6-13)
	sn       uint32     // stream serial number (14-17)
	seqNo    uint32     // page sequence no (18-21)
	checksum uint32     // page checksum (22-25)
	count    uint8      // page_segments (26)
	segments [256]uint8 // segment_table (27-n, n=page_segments+26)
}

func (ph *pageHeader) Len() int {
	return int(27 + ph.count)
}

func (ph *pageHeader) PayloadLen() int {
	l := 0
	for i := uint8(0); i < ph.count; i++ {
		l += int(ph.segments[i])
	}
	return l
}

// caller should ensure buf should be a least 27 bytes
func (ph *pageHeader) ReadHeader(p []byte) error {
	if len(p) < 27 {
		return errors.New("Invalid buffer size.")
	}
	if !bytes.Equal(p[:4], capturePattern) {
		return errors.New("Capture pattern not found.")
	}
	ph.version = p[4]
	if ph.version != 0 {
		return fmt.Errorf("Unsupported version: %d", ph.version)
	}
	ph.flags = p[5]
	ph.pos = binary.LittleEndian.Uint64(p[6:])
	ph.sn = binary.LittleEndian.Uint32(p[14:])
	ph.seqNo = binary.LittleEndian.Uint32(p[18:])
	ph.checksum = binary.LittleEndian.Uint32(p[22:])
	ph.count = p[26]
	if ph.count > 255 {
		return errors.New("Invalid packet count %d.", ph.count)
	}
	return nil
}

func (ph *pageHeader) ReadSegments(p []byte) error {
	if len(p) < ph.count {
		return errors.New("Invalid buffer size.")
	}
	for i := uint8(0); i < ph.count; i++ {
		ph.segments[i] = p[i]
	}
	return nil
}

const (
	flagCodecVideo = 1 << iota
	flagCodecAudio = 1 << iota
	flagCodecText  = 1 << iota
)

type codecInfo struct {
	prefix []byte
	name   string
	flag   int
}

// http://wiki.xiph.org/index.php/MIMETypesCodecs
var codecTable = []codecInfo{
	{[]byte("CELT    "), "celt", flagCodecAudio},
	{[]byte("CMML\x00\x00\x00\x00"), "cmml", flagCodecText},
	{[]byte("BBCD\x00"), "dirac", flagCodecVideo},
	{[]byte("\177FLAC"), "flac", flagCodecAudio},
	{[]byte("\213JNG\r\n\032\n"), "jng", flagCodecVideo},
	{[]byte("\x80kate\x00\x00\x00"), "kate", flagCodecText},
	{[]byte("OggMIDI\x00"), "midi", flagCodecText},
	{[]byte("\212MNG\r\n\032\n"), "mng", flagCodecVideo},
	{[]byte("OpusHead"), "opus", flagCodecAudio},
	{[]byte("PCM     "), "pcm", flagCodecAudio},
	{[]byte("\211PNG\r\n\032\n"), "png", flagCodecVideo},
	{[]byte("Speex   "), "speex", flagCodecAudio},
	{[]byte("\x80theora"), "theora", flagCodecVideo},
	{[]byte("\x01vorbis"), "vorbis", flagCodecAudio},
	{[]byte("YUV4MPEG"), "yuv4mpeg", flagCodecVideo},
}

type streamInfo struct {
	sn    uint32
	codec *codecInfo
}

func findCodecInfo(hdr []byte) *codecInfo {
	hlen := len(hdr)
	for _, pte := range codecTable {
		plen := len(pte.prefix)
		if hlen > plen && bytes.Equal(hdr[:plen], pte.prefix) {
			return &pte
		}
	}
	return nil
}

// peek looks at the first pages for every streams and extract stream info.
func peek(in io.ReadSeeker) ([]streamInfo, error) {
	buf := make([]byte, 256)
	hbuf := buf[:27]
	streams := make([]streamInfo, 0, 3)

	ph := new(pageHeader)
	for {
		offset := 0

		_, err := io.ReadFull(in, hbuf)
		if err != nil {
			return streams, err
		}
		err = ph.ReadHeader(hbuf)
		if err != nil {
			return streams, err
		}

		segbuf := buf[:ph.count]
		_, err = io.ReadFull(in, segbuf)
		if err != nil {
			return streams, err
		}
		ph.ReadSegments(segbuf)
		if ph.flags&flagFirst < 1 {
			return streams, nil
		}

		offset += ph.PayloadLen()
		if ph.count > 0 {
			size := int(ph.segments[0])
			_, err := io.ReadFull(in, buf[:size])
			if err != nil {
				return streams, err
			}
			codec := findCodecInfo(buf)
			if codec != nil {
				streams = append(streams, streamInfo{ph.sn, codec})
			}
			offset -= size
		}

		_, err = in.Seek(int64(offset), 1)
		if err != nil {
			return streams, err
		}
	}
}

func mimeType(streams []streamInfo) string {
	flags := 0
	for _, stream := range streams {
		flags |= stream.codec.flag
	}
	if flags&flagCodecVideo > 0 {
		return "video/ogg"
	}
	if flags&flagCodecAudio > 0 {
		return "audio/ogg"
	}
	return "application/ogg"
}

func contentType(streams []streamInfo) string {
	flags := 0
	for _, stream := range streams {
		flags |= stream.codec.flag
	}
	if flags&flagCodecVideo > 0 {
		ct := bytes.NewBufferString("video/ogg; codecs=\"")
		for _, stream := range streams {
			if stream.codec.flag&flagCodecVideo > 0 {
				ct.WriteString(stream.codec.name)
				break
			}
		}
		for _, stream := range streams {
			if stream.codec.flag&flagCodecAudio > 0 {
				ct.WriteString(", ")
				ct.WriteString(stream.codec.name)
				break
			}
		}
		ct.WriteRune('"')
		return ct.String()
	}
	if flags&flagCodecAudio > 0 {
		ct := bytes.NewBufferString("audio/ogg; codecs=\"")
		for _, stream := range streams {
			if stream.codec.flag&flagCodecAudio > 0 {
				ct.WriteString(stream.codec.name)
				break
			}
		}
		ct.WriteRune('"')
		return ct.String()
	}
	return "application/ogg"
}

func main() {
	f, err := os.Open("/Users/gwik/dev/go/src/camlistore.org/small.ogv")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	buf := make([]byte, 512)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		os.Exit(128)
	}
	streams, err := peek(bytes.NewReader(buf))
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	for _, stream := range streams {
		fmt.Printf("stream sn=%d codec=%s\n", stream.sn, stream.codec.name)
	}

	fmt.Println(mimeType(streams))
	fmt.Println(contentType(streams))
}
