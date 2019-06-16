// Copyright (C) 2017, 2019  Steve Merrony

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
	"fmt"
	"time"

	"github.com/distributed/sers"
)

const breakMs = 110 // How many ms to hold BREAK signal

var (
	serPort              sers.SerialPort // io.ReadWriteCloser
	sendSerialBreakChan  chan bool
	stopSerialWriterChan chan bool
)

func openSerialPort(port string, baud int, bits int, parityStr string, stopBits int) bool {
	var parity int
	switch parityStr {
	case "None":
		parity = sers.N
	case "Even":
		parity = sers.E
	case "Odd":
		parity = sers.O
	}
	serPort, err = sers.Open(port) // serial.Open(options)
	if err != nil {
		fmt.Printf("ERROR: Could not open serial port - %s\n", err.Error())
		return false
	}
	if err = serPort.SetMode(baud, bits, parity, stopBits, sers.NO_HANDSHAKE); err != nil {
		fmt.Printf("ERROR: Could not set serial part mode as requested - %s\n", err.Error())
		return false
	}
	sendSerialBreakChan = make(chan bool)
	stopSerialWriterChan = make(chan bool)
	go serialReader(serPort, fromHostChan)
	go serialWriter(serPort, keyboardChan)
	terminal.rwMutex.Lock()
	terminal.connected = serialConnected
	terminal.serialPort = port
	terminal.rwMutex.Unlock()
	return true
}

func closeSerialPort() {
	serPort.Close()
	sendSerialBreakChan = nil
	stopSerialWriterChan <- true
	terminal.rwMutex.Lock()
	terminal.connected = disconnected
	terminal.rwMutex.Unlock()
}

func serialReader(port sers.SerialPort, hostChan chan []byte) {
	for {
		hostBytes := make([]byte, hostBuffSize)
		n, err := port.Read(hostBytes)
		if n == 0 {
			fmt.Println("WARNING: serialReader got zero length message")
			if err == nil {
				continue
			}
		}
		if err != nil {
			fmt.Printf("ERROR: Could not read from Serial Port - %s\n", err.Error())
			stopSerialWriterChan <- true
			return
		}
		hostChan <- hostBytes[:n]
	}
}

func serialWriter(port sers.SerialPort, kbdChan chan byte) {
	for {
		select {
		case k := <-kbdChan:
			port.Write([]byte{k})
		case sb := <-sendSerialBreakChan:
			if sb {
				//fmt.Println("DEBUG: Setting BREAK on")
				port.SetBreak(true)
				time.Sleep(breakMs * time.Millisecond)
				port.SetBreak(false)
				//fmt.Println("DEBUG: Set BREAK off")
			}
		case <-stopSerialWriterChan:
			fmt.Println("INFO: serialWriter stopping")
			return
		}
	}
}
