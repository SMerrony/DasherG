package main

import "os"

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
	holding, logging                   bool
	connected                          int
	emulation                          int
	logFile                            *os.File
}

func (s *Status) setup() {
	s.visCols = defaultCols
	s.visLines = defaultLines
	s.emulation = d200
}
