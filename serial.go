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
	"fmt"
	"io"
	"log"

	"github.com/jacobsa/go-serial/serial"
)

var (
	serPort              io.ReadWriteCloser
	stopSerialWriterChan = make(chan bool)
)

func openSerialPort(port string, baud int, bits int, parityStr string, stopBits int) bool {
	var parity serial.ParityMode
	switch parityStr {
	case "None":
		parity = serial.PARITY_NONE
	case "Even":
		parity = serial.PARITY_EVEN
	case "Odd":
		parity = serial.PARITY_ODD
	}
	options := serial.OpenOptions{
		PortName:        port,
		BaudRate:        uint(baud),
		DataBits:        uint(bits),
		StopBits:        uint(stopBits),
		ParityMode:      parity,
		MinimumReadSize: 1,
	}
	serPort, err = serial.Open(options)
	if err != nil {
		fmt.Printf("ERROR opening serial port %v\n", err)
		return false
	}
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
	stopSerialWriterChan <- true
	terminal.rwMutex.Lock()
	terminal.connected = disconnected
	terminal.rwMutex.Unlock()
}

func serialReader(port io.ReadWriteCloser, hostChan chan []byte) {
	for {
		hostBytes := make([]byte, hostBuffSize)
		n, err := port.Read(hostBytes)
		if n == 0 {
			fmt.Println("serialReader got zero length message, stopping")
			closeSerial()
			return
		}
		if err != nil {
			log.Fatal("serialReader got errror reading from port ", err.Error())
		}
		hostChan <- hostBytes[:n]
	}

}

func serialWriter(port io.ReadWriteCloser, kbdChan chan byte) {
	for {
		select {
		case k := <-kbdChan:
			port.Write([]byte{k})
			//fmt.Printf("Wrote <%d> to host\n", k)
		case <-stopSerialWriterChan:
			fmt.Println("serialWriter stopping")
			return
		}
	}
}
