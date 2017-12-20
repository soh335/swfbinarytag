package swfbinarytag

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"io/ioutil"
	"math"

	"github.com/pkg/errors"
)

var (
	OverflowErr = errors.New("overflow")
	NotfoundErr = errors.New("not found")
)

const (
	binaryType = 87
)

const (
	unknowncompreesed = 0
	uncompreesed      = 0x46
	zlibcompressed    = 0x43
)

type tag struct {
	tagType uint16
	content []byte
}

type src struct {
	input []byte
	pos   int
}

func (s *src) eof() bool {
	return s.pos >= len(s.input)
}

func (s *src) seekUI8() error {
	if s.pos+1 > len(s.input) {
		return OverflowErr
	}
	s.pos = s.pos + 1
	return nil
}

func (s *src) seekUI16() error {
	if s.pos+2 > len(s.input) {
		return OverflowErr
	}
	s.pos = s.pos + 2
	return nil
}

func (s *src) seekUI32() error {
	if s.pos+4 > len(s.input) {
		return OverflowErr
	}
	s.pos = s.pos + 4
	return nil
}

func (s *src) seekRect() error {
	if s.pos+1 > len(s.input) {
		return OverflowErr
	}

	b := s.input[s.pos]
	size := (b >> (8 - 5)) & ((1 << 5) - 1)
	sumBits := 5 + size*4
	seekBytes := int(math.Ceil(float64(sumBits) / 8))

	if s.pos+seekBytes > len(s.input) {
		return OverflowErr
	}

	s.pos = s.pos + seekBytes

	return nil
}

func (s *src) read(u int) ([]byte, error) {
	if s.pos+u > len(s.input) {
		return nil, OverflowErr
	}
	byts := s.input[s.pos : s.pos+u]
	s.pos = s.pos + u
	return byts, nil
}

func (s *src) readUI8() (uint8, error) {
	if s.pos+1 > len(s.input) {
		return 0, OverflowErr
	}

	byt := s.input[s.pos]
	s.pos = s.pos + 1
	return byt, nil
}

func (s *src) readUI16() (uint16, error) {
	if s.pos+2 > len(s.input) {
		return 0, OverflowErr
	}

	byts := s.input[s.pos : s.pos+2]
	s.pos = s.pos + 2
	return binary.LittleEndian.Uint16(byts), nil
}

func (s *src) readUI32() (uint32, error) {
	if s.pos+4 > len(s.input) {
		return 0, OverflowErr
	}

	byts := s.input[s.pos : s.pos+4]
	s.pos = s.pos + 4
	return binary.LittleEndian.Uint32(byts), nil
}

func Find(r io.Reader, id uint16) ([]byte, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}
	var s src
	s.input = input
	compreessed, err := parseHeader1(&s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parseHeader")
	}
	switch compreessed {
	case zlibcompressed:
		zr, err := zlib.NewReader(bytes.NewReader(s.input[s.pos:]))
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize zlib reader")
		}
		input, err := ioutil.ReadAll(zr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read zlib data")
		}
		s = src{}
		s.input = input
	case uncompreesed:
		break
	default:
		return nil, errors.New("unknown type")
	}

	for {
		tag, err := parseTag(&s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parseTag")
		}
		switch tag.tagType {
		case binaryType:
			binaryTag, err := parseBinaryTag(tag)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read parse binary tag")
			}
			if binaryTag.Tag == id {
				return binaryTag.Data, nil
			}
		}

		if s.eof() {
			break
		}
	}

	return nil, NotfoundErr
}

func parseHeader1(s *src) (uint8, error) {
	// signature: ui8
	sig1, err := s.readUI8()
	if err != nil {
		return unknowncompreesed, errors.Wrap(err, "failed to read signature")
	}

	switch sig1 {
	case uncompreesed, zlibcompressed:
		break
	default:
		return unknowncompreesed, errors.Errorf("unsupport signature: %d", sig1)
	}

	// signature: ui8
	if err := s.seekUI8(); err != nil {
		return unknowncompreesed, errors.Wrap(err, "failed to seek signature")
	}

	// signature: ui8
	if err := s.seekUI8(); err != nil {
		return unknowncompreesed, errors.Wrap(err, "failed to seek signature")
	}

	// version: ui8
	if err := s.seekUI8(); err != nil {
		return unknowncompreesed, errors.Wrap(err, "failed to seek version")
	}

	// file length: ui32
	if err := s.seekUI32(); err != nil {
		return unknowncompreesed, errors.Wrap(err, "failed to seek file length")
	}

	return sig1, nil
}

func parseHeader2(s *src) error {
	// frame size: rect
	if err := s.seekRect(); err != nil {
		return errors.Wrap(err, "failed to seek frame size")
	}

	// frame rate: ui16
	if err := s.seekUI16(); err != nil {
		return errors.Wrap(err, "failed to seek frame rate")
	}

	// frame count: ui16
	if err := s.seekUI16(); err != nil {
		return errors.Wrap(err, "failed to seek frame count")
	}

	return nil
}

func parseTag(s *src) (*tag, error) {
	tagCodeAndLength, err := s.readUI16()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read tagCodeAndLength")
	}

	// first 10 bits is tagcode, after 6 bits length
	// if length is 0x3f, we should parse as long tag

	tagType := tagCodeAndLength >> 6
	length := uint32(tagCodeAndLength & 0x3f)

	// its mean long tag
	if length == 0x3f {
		length, err = s.readUI32()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read length")
		}
	}

	content, err := s.read(int(length))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read tag reamin content")
	}

	t := &tag{
		tagType: tagType,
		content: content,
	}

	return t, nil
}

type defineBinaryDataTag struct {
	TagType  uint16
	Length   int
	Tag      uint16
	Reserved uint32
	Data     []byte
}

func parseBinaryTag(t *tag) (*defineBinaryDataTag, error) {
	s := &src{input: t.content}

	// confusing
	tag, err := s.readUI16()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read tag")
	}
	if err := s.seekUI32(); err != nil {
		return nil, errors.Wrap(err, "failed to read reserved field")
	}
	data := s.input[s.pos:]

	return &defineBinaryDataTag{
		TagType: t.tagType,
		Length:  len(t.content),
		Tag:     tag,
		Data:    data,
	}, nil

}
