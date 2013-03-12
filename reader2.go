
package mp4

import (
	"github.com/go-av/av"
	"os"
	"errors"
)

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
	//	l.Printf(" at %d pos %f\n", m, a[m].pos)
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

func testSearchIndex() {
	a := make([]mp4index, 10)
	for i, _ := range a {
		a[i].pos = float32(i)
	}
	for i := -4; i < 14; i++ {
		pos := float32(i)+0.1
		l.Printf("search(%f)\n", pos)
		r := binSearch(a, pos)
		l.Printf(" =%d\n", r)
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

func (m *mp4) closeReader() {
	m.rat.Close()
}

