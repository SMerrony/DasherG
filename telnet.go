package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	telnetCmdSE   = 240
	telnetCmdNOP  = 241
	telnetCmdDM   = 242
	telnetCmdBRK  = 243
	telnetCmdIP   = 244
	telnetCmdAO   = 245
	telnetCmdAYT  = 246
	telnetCmdEC   = 247
	telnetCmdEL   = 248
	telnetCmdGA   = 249
	telnetCmdSB   = 250
	telnetCmdWILL = 251
	telnetCmdWONT = 252
	telnetCmdDO   = 253
	telnetCmdDONT = 254
	telnetCmdIAC  = 255

	telnetOptBIN    = 0
	telnetOptECHO   = 1
	telnetOptRECON  = 2
	telnetOptSGA    = 3
	telnetOptSTATUS = 5
	telnetOptCOLS   = 8
	telnetOptROWS   = 9
	telnetOptEASCII = 17
	telnetOptLOGOUT = 18
	telnetOptTTYPE  = 24
	telnetOptNAWS   = 31 // window size
	telnetOptTSPEED = 32
	telnetOptXDISP  = 35
	telnetOptNEWENV = 39

	dialTimeout = time.Second * 10
)

func openTelnetConn(hostName string, portNum int) bool {
	hostString := hostName + ":" + strconv.Itoa(portNum)
	conn, err := net.DialTimeout("tcp", hostString, dialTimeout)
	if err != nil {
		return false
	}
	go telnetReader(bufio.NewReader(conn), hostChan)
	go telnetWriter(bufio.NewWriter(conn), keyboardChan)
	return true
}

func telnetReader(reader *bufio.Reader, hostChan chan []byte) {
	hostBytes := make([]byte, hostBuffSize)
	for {
		n, _ := reader.Read(hostBytes)
		hostChan <- hostBytes[:n]
	}
}

func telnetWriter(writer *bufio.Writer, kbdChan chan byte) {
	for k := range kbdChan {
		writer.Write([]byte{k})
		fmt.Printf("Wrote <%d> to host\n", k)
	}
}
