
package mp4

import (
	"io"
	"log"
	"os"
	"bytes"
)

type dummyWriter struct {
}

func (m *dummyWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

var (
	dummyW = &dummyWriter{}
	l = log.New(dummyW, "mp4: ", 0)
)

type mp4trk struct {
	cc4 string
	keyFrames []int
	sampleSizes []int
	chunkOffs []int64
	stts []mp4stts
	stsc []mp4stsc
	index []mp4index
	extra []byte
	timeScale int
	dur int
	i int
	codec, idx int
}

type mp4stsc struct {
	first, cnt, id int
}

type mp4index struct {
	ts, size int
	off int64
	key bool
	pos float32
}

type mp4stts struct {
	cnt, dur int
}

type mp4 struct {
	trk []*mp4trk
	vtrk, atrk *mp4trk
	Dur, Pos float32
	W, H int
	rat *os.File
	AACCfg []byte
	PPS []byte

	mdatOff int64
	w, w2 *os.File
	tmp, path string
}

func ReadUint(r io.Reader, n int) (ret uint, err error) {
	b, err := ReadBuf(r, n)
	for i := 0; i < n; i++ {
		ret <<= 8
		ret += uint(b[i])
	}
	return
}

func ReadInt(r io.Reader, n int) (ret int, err error) {
	_ret, err := ReadUint(r, n)
	ret = int(_ret)
	return
}

func WriteInt(r io.Writer, v int, n int) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[n-i-1] = byte(v&0xff)
		v >>= 8
	}
	r.Write(b)
}

func WriteString(w io.Writer, str string) {
	w.Write([]byte(str))
}

func WriteTag(w io.Writer, tag string, cb func(w io.Writer)) {
	var b bytes.Buffer
	cb(&b)
	WriteInt(w, b.Len()+8, 4)
	WriteString(w, tag)
	w.Write(b.Bytes())
}

func ReadAll(r io.Reader) ([]byte) {
	var b bytes.Buffer
	io.Copy(&b, r)
	return b.Bytes()
}

func ReadBuf(r io.Reader, n int) ([]byte, error) {
	b := make([]byte, n)
	n, err := r.Read(b)
	return b, err
}

func ReadString(r io.Reader, n int) (string, error) {
	b, err := ReadBuf(r, n)
	return string(b), err
}

func LogLevel(i int) {
	if (i > 0) {
		l = log.New(os.Stderr, "mp4: ", 0)
	}
}

func (m *mp4) Close() {
	if m.rat != nil {
		m.closeReader()
	} else {
		m.closeWriter()
	}
}

