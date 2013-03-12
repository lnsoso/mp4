
package mp4

import (
	"github.com/go-av/av"
	"io"
	"log"
	"os"
	"errors"
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
	rat io.ReaderAt
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

func binSearch(a []mp4index, pos float32) int {
	l := 0
	r := len(a)-1
	for ; l < r-1; {
		m := (l + r) / 2
		if pos < a[m].pos {
			r = m
		} else {
			l = m
		}
	//	log.Printf(" at %d pos %f\n", m, a[m].pos)
	}
	return l
}

func searchIndex(pos float32, trk *mp4trk, key bool) (ret int) {
	if key {
		a := trk.keyFrames
		b := trk.index
		for i := 0; i < len(a)-1; i++ {
			if b[a[i]-1].pos < pos && pos < b[a[i+1]-1].pos {
				ret = a[i]-1
				return
			}
		}
	} else {
		ret = binSearch(trk.index, pos)
	}
	return
}

func TestSearchIndex() {
	a := make([]mp4index, 10)
	for i, _ := range a {
		a[i].pos = float32(i)
	}
	for i := -4; i < 14; i++ {
		pos := float32(i)+0.1
		log.Printf("search(%f)\n", pos)
		r := binSearch(a, pos)
		log.Printf(" =%d\n", r)
	}
}

func (m *mp4) SeekKey(pos float32) {
	l.Printf("SeekKey %f\n", pos)
	m.vtrk.i = searchIndex(pos, m.vtrk, true)
	l.Printf(" V: %f\n", m.vtrk.index[m.vtrk.i].pos)
	if m.atrk != nil {
		m.atrk.i = searchIndex(pos, m.atrk, false)
		l.Printf(" A: %f\n", m.atrk.index[m.atrk.i].pos)
	}
}

func (m *mp4) readTo(trks []*mp4trk, end float32) (ret []*av.Packet, pos float32) {
	for {
		var mt *mp4trk
		for _, t := range trks {
			if t.i >= len(t.index) {
				continue
			}
			if mt == nil || t.index[t.i].pos < mt.index[mt.i].pos {
				mt = t
			}
		}
		if mt == nil {
			l.Printf("mt == nil\n")
			break
		}
		pos = mt.index[mt.i].pos
		if pos >= end {
			break
		}
		b := make([]byte, mt.index[mt.i].size)
		m.rat.ReadAt(b, mt.index[mt.i].off)
		ret = append(ret, &av.Packet{
			Codec:mt.codec, Key:mt.index[mt.i].key,
			Pos:mt.index[mt.i].pos, Data:b,
		})
		mt.i++
	}
	return
}

func (m *mp4) ReadDur(dur float32) (ret []*av.Packet) {
	l.Printf("ReadDur %f\n", dur)
	ret, m.Pos = m.readTo([]*mp4trk{m.vtrk, m.atrk}, m.Pos + dur)
	l.Printf(" got %d packets\n", len(ret))
	return
}

func Open(path string) (m *mp4, err error) {
	m = &mp4{}
	r, err := os.Open(path)
	if err != nil {
		return
	}
	m.rat = r
	m.readAtom(r, 0, nil)
	for _, t := range m.trk {
		m.parseTrk(t)
	}
	if m.vtrk == nil {
		err = errors.New("no video track")
		return
	}
	m.Dur = float32(m.vtrk.dur) / float32(m.vtrk.timeScale)
	return
}

func LogLevel(i int) {
	if (i > 0) {
		l = log.New(os.Stderr, "mp4: ", 0)
	}
}

