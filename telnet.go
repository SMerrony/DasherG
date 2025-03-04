// Copyright (C) 2017,2019  Steve Merrony

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

	// telnetOptBIN    = 0
	// telnetOptECHO   = 1
	// telnetOptRECON  = 2
	// telnetOptSGA    = 3
	// telnetOptSTATUS = 5
	// telnetOptCOLS   = 8
	// telnetOptROWS   = 9
	// telnetOptEASCII = 17
	// telnetOptLOGOUT = 18
	// telnetOptTTYPE  = 24
	// telnetOptNAWS   = 31 // window size
	// telnetOptTSPEED = 32
	// telnetOptXDISP  = 35
	// telnetOptNEWENV = 39

	dialTimeout = time.Second * 10
)

type telnetSessionT struct {
	conn                 net.Conn
	stopTelnetWriterChan chan bool
}

func newTelnetSession() *telnetSessionT {
	sess := new(telnetSessionT)
	sess.stopTelnetWriterChan = make(chan bool)
	return sess
}

func (sess *telnetSessionT) openTelnetConn(hostName string, portNum int) bool {
	var err error
	hostString := hostName + ":" + strconv.Itoa(portNum)
	sess.conn, err = net.DialTimeout("tcp", hostString, dialTimeout)
	if err != nil {
		return false
	}
	go sess.telnetReader(fromHostChan)
	go sess.telnetWriter(keyboardChan)
	terminal.rwMutex.Lock()
	terminal.connectionType = telnetConnected
	terminal.remoteHost = hostName
	terminal.remotePort = strconv.Itoa(portNum)
	terminal.rwMutex.Unlock()
	return true
}

func (sess *telnetSessionT) closeTelnetConn() {
	sess.conn.Close()
	select {
	case sess.stopTelnetWriterChan <- true:
	default:
	}
	terminal.rwMutex.Lock()
	terminal.connectionType = disconnected
	terminal.rwMutex.Unlock()
}

func (sess *telnetSessionT) telnetReader(hostChan chan<- []byte) {
	fmt.Println("INFO: telnetReader starting")
	for {
		hostBytes := make([]byte, hostBuffSize)
		n, err := sess.conn.Read(hostBytes)
		if n == 0 {
			//log.Fatalf("telnet got zero-byte message from host")
			fmt.Println("INFO: telnetReader got zero length message, stopping")
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

func (sess *telnetSessionT) telnetWriter(kbdChan <-chan byte) {
	fmt.Println("INFO: telnetWriter starting")
	writer := bufio.NewWriter(sess.conn)
	for {
		select {
		case k := <-kbdChan:
			writer.Write([]byte{k})
			writer.Flush()
			//fmt.Printf("Wrote <%d> to host\n", k)
		case <-sess.stopTelnetWriterChan:
			fmt.Println("INFO: telnetWriter stopping")
			return
		}
	}
}
