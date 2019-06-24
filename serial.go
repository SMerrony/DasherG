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
	"strconv"
	"time"

	"github.com/distributed/sers"
)

const breakMs = 110 // How many ms to hold BREAK signal

type serialSessionT struct {
	serPort              sers.SerialPort
	sendSerialBreakChan  chan bool
	stopSerialWriterChan chan bool
}

func newSerialSession() *serialSessionT {
	ser := new(serialSessionT)
	ser.sendSerialBreakChan = make(chan bool)
	ser.stopSerialWriterChan = make(chan bool, 2)
	return ser
}

func (ser *serialSessionT) openSerialPort(port string, baud int, bits int, parityStr string, stopBits int) bool {
	var parity int
	switch parityStr {
	case "None":
		parity = sers.N
	case "Even":
		parity = sers.E
	case "Odd":
		parity = sers.O
	}
	ser.serPort, err = sers.Open(port) // serial.Open(options)
	if err != nil {
		fmt.Printf("ERROR: Could not open serial port - %s\n", err.Error())
		return false
	}
	if err = ser.serPort.SetMode(baud, bits, parity, stopBits, sers.NO_HANDSHAKE); err != nil {
		fmt.Printf("ERROR: Could not set serial part mode as requested - %s\n", err.Error())
		return false
	}

	go ser.serialReader(fromHostChan)
	go ser.serialWriter(keyboardChan)
	terminal.rwMutex.Lock()
	terminal.connectionType = serialConnected
	terminal.serialPort = port
	terminal.serialBaud = strconv.Itoa(baud)
	terminal.serialBits = strconv.Itoa(bits)
	terminal.serialParity = string(parityStr[0])
	terminal.serialStopBits = strconv.Itoa(stopBits)
	terminal.rwMutex.Unlock()
	return true
}

func (ser *serialSessionT) closeSerialPort() {
	ser.serPort.Close()
	//sendSerialBreakChan = nil
	ser.stopSerialWriterChan <- true
	terminal.rwMutex.Lock()
	terminal.connectionType = disconnected
	terminal.rwMutex.Unlock()
}

func (ser *serialSessionT) serialReader(hostChan chan []byte) {
	for {
		hostBytes := make([]byte, hostBuffSize)
		n, err := ser.serPort.Read(hostBytes)
		if n == 0 {
			fmt.Println("WARNING: serialReader got zero length message")
			if err == nil {
				continue
			}
		}
		if err != nil {
			fmt.Printf("WARNING: Could not read from Serial Port - %s\n", err.Error())
			fmt.Println("INFO: Stopping serialReader and asking serialWtiter to stop")
			ser.stopSerialWriterChan <- true
			return
		}
		hostChan <- hostBytes[:n]
	}
}

func (ser *serialSessionT) serialWriter(kbdChan chan byte) {
	// drain stop chan in case of multiple stops queued
	for len(ser.stopSerialWriterChan) > 0 {
		<-ser.stopSerialWriterChan
	}
	// loop
	for {
		select {
		case k := <-kbdChan:
			ser.serPort.Write([]byte{k})
		case sb := <-ser.sendSerialBreakChan:
			if sb {
				ser.serPort.SetBreak(true)
				time.Sleep(breakMs * time.Millisecond)
				ser.serPort.SetBreak(false)
			}
		case <-ser.stopSerialWriterChan:
			fmt.Println("INFO: serialWriter stopping")
			return
		}
	}
}
