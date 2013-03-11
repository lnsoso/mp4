mp4
====

Golang mp4 reader and writer

Reader example

	m, _ := mp4.Open("test.mp4")
	log.Printf("video %dx%d", m.W, m.H)
	log.Printf("duration %.2fs", m.Dur)
	m.SeekKey(32.0) // seek to nearest keyframe
	pkts := m.ReadDur(2.0) // read packets in 2s

Writer example

	m, _ := mp4.Create("out.mp4")
	m.WriteAAC(pkt)
	m.WriteH264(pkt)
	m.Close()

