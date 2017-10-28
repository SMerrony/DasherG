package main

const (
	disconnected    = 0
	serialConnected = 1
	telnetConnected = 2

	d200 = 200
	d210 = 210
	d211 = 211
)

type Status struct {
	visLines, visCols                  int
	serialPort, remoteHost, remotePort string
	holding                            bool
	connection                         int
	emulation                          int
}

func (s *Status) setup() {
	s.visCols = defaultCols
	s.visLines = defaultLines
}
