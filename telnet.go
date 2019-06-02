// Copyright (C) 2017  Steve Merrony

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"bufio"
	"fmt"
	"log"
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

var (
	conn                 net.Conn
	err                  error
	stopTelnetWriterChan = make(chan bool)
	lastHost             string
	lastPort             int
)

func openTelnetConn(hostName string, portNum int) bool {
	hostString := hostName + ":" + strconv.Itoa(portNum)
	conn, err = net.DialTimeout("tcp", hostString, dialTimeout)
	if err != nil {
		return false
	}
	lastHost = hostName
	lastPort = portNum
	go telnetReader(conn, fromHostChan)
	go telnetWriter(bufio.NewWriter(conn), keyboardChan)
	terminal.rwMutex.Lock()
	terminal.connected = telnetConnected
	terminal.remoteHost = hostName
	terminal.remotePort = strconv.Itoa(portNum)
	terminal.rwMutex.Unlock()
	return true
}

func closeTelnetConn() {
	conn.Close()
	stopTelnetWriterChan <- true
	terminal.rwMutex.Lock()
	terminal.connected = disconnected
	terminal.rwMutex.Unlock()
}

func telnetReader(con net.Conn, hostChan chan<- []byte) {
	for {
		hostBytes := make([]byte, hostBuffSize)
		n, err := con.Read(hostBytes)
		if n == 0 {
			//log.Fatalf("telnet got zero-byte message from host")
			fmt.Println("telnetReader got zero length message, stopping")
			telnetClose()
			return
		}
		if err != nil {
			log.Fatal("telnetReader got errror reading from host ", err.Error())
		}
		//fmt.Printf("telentReader got <%s> from host\n", hostBytes)
		hostChan <- hostBytes[:n]
	}
}

func telnetWriter(writer *bufio.Writer, kbdChan <-chan byte) {
	for {
		select {
		case k := <-kbdChan:
			writer.Write([]byte{k})
			writer.Flush()
			//fmt.Printf("Wrote <%d> to host\n", k)
		case <-stopTelnetWriterChan:
			fmt.Println("telnetWriter stopping")
			return
		}
	}
}
