// This is based on "chriszzzzz"'s fork of Omegaice's go-xmodem code

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, asciiSUBlicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, asciiSUBject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or asciiSUBstantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

const asciiSOH byte = 0x01
const asciiSTX byte = 0x02
const asciiEOT byte = 0x04
const asciiACK byte = 0x06
const asciiNAK byte = 0x15
const asciiCAN byte = 0x18
const asciiSUB byte = 0x1a
const xmodemPOLL byte = 'C'

const xmodemShortPacketLen = 128
const xmodemLongPacketLen = 1024

const xmodemDebugFlag = true

func crc16(data []byte) uint16 {
	var u16CRC uint16

	for _, character := range data {
		part := uint16(character)

		u16CRC = u16CRC ^ (part << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	return u16CRC
}

func crc16Constant(data []byte, length int) uint16 {
	var u16CRC uint16

	for _, character := range data {
		part := uint16(character)

		u16CRC = u16CRC ^ (part << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	for c := 0; c < length-len(data); c++ {
		u16CRC = u16CRC ^ (0x04 << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	return u16CRC
}

func sendBlock(tx chan byte, block int, data []byte, packetPayloadLen int) error {
	startByte := asciiSOH
	if packetPayloadLen == xmodemLongPacketLen {
		startByte = asciiSTX
	}
	// send start byte and length
	if xmodemDebugFlag {
		fmt.Printf("DEBUG: Sending start byte and length of %d bytes\n", block)
	}
	tx <- startByte
	blockNum := byte(uint8(block % 256))
	tx <- blockNum
	tx <- ^blockNum

	//send data
	var toSend bytes.Buffer
	toSend.Write(data)
	for padding := len(data); padding < packetPayloadLen; padding++ {
		toSend.Write([]byte{asciiEOT})
	}

	if xmodemDebugFlag {
		fmt.Printf("DEBUG: Sending block: %d\n", block)
	}
	for sent := 0; sent < packetPayloadLen; sent++ {
		if xmodemDebugFlag {
			fmt.Printf("DEBUG: Sending byte: %d of packet: %d\n", sent, block)
		}
		tx <- toSend.Bytes()[sent]
	}
	//calc CRC
	u16CRC := crc16Constant(data, packetPayloadLen)
	if xmodemDebugFlag {
		fmt.Println("DEBUG: Sending CRC")
	}
	//send CRC
	tx <- byte(uint8(u16CRC >> 8))
	tx <- byte(uint8(u16CRC & 0x0FF))

	return nil
}

// XmodemSendShort transmits a file via XMODEM-CRC using the short (128-byte) packet length
func XmodemSendShort(rx chan byte, tx chan byte, f *os.File) error {
	return xmodemSend(rx, tx, f, xmodemShortPacketLen)
}

// XmodemSendLong transmits a file via XMODEM-CRC using the long (1024-byte) packet length
func XmodemSendLong(rx chan byte, tx chan byte, f *os.File) error {
	return xmodemSend(rx, tx, f, xmodemLongPacketLen)
}

func xmodemSend(rx chan byte, tx chan byte, f *os.File, packetPayloadLen int) error {

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.New("XMODEM Could not read file to send")
	}
	if xmodemDebugFlag {
		fmt.Printf("XMODEM: Read %d bytes from file to transmit\n", len(data))
	}

	// oBuffer := make([]byte, 1)
	if xmodemDebugFlag {
		fmt.Println("XMODEM: Waiting for POLL")
	}
	if <-rx == xmodemPOLL {
		if xmodemDebugFlag {
			fmt.Println("XMODEM: Got POLL")
		}
		var blocks = len(data) / packetPayloadLen
		if len(data) > blocks*packetPayloadLen {
			blocks++
		}
		if xmodemDebugFlag {
			fmt.Printf("XMODEM: Total blocks to send: %d\n", blocks)
		}
		failed := 0
		var currentBlock = 0
		for currentBlock < blocks && failed < 10 {
			sendBlock(tx, currentBlock+1, data[currentBlock*packetPayloadLen:(currentBlock+1)*packetPayloadLen], packetPayloadLen)
			if xmodemDebugFlag {
				fmt.Println("XMODEM: sendBlock complete, waiting for response...")
			}
			resp := <-rx
			switch resp {
			case asciiACK:
				currentBlock++
				if xmodemDebugFlag {
					fmt.Printf("XMODEM: Block: %d ACKed\n", currentBlock)
				}
				failed = 0
			case asciiNAK:
				failed++
				if xmodemDebugFlag {
					fmt.Printf("XMODEM: Block: %d NAKed\n", currentBlock)
				}
			default:
				fmt.Printf("XMODEM: Unexpected response to packet, got: 0x%x\n", resp)
				return errors.New("XMODEM: Unexpected response to packet")
			}
		}

		if failed == 10 {
			return errors.New("XMODEM: Send failed - too many retries")
		}

		tx <- asciiEOT
	}

	return nil
}

// XModemReceive received a file using the XMODEM-CRC protocol
// in either 128 or 1024-byte packets as determined by the sender.
func XModemReceive(rx chan byte, tx chan byte) ([]byte, error) {
	var (
		data       bytes.Buffer
		packetSize int
		crc        uint16
	)
	if xmodemDebugFlag {
		fmt.Println("XMODEM: Sending POLL")
	}

	// Start Connection
	tx <- xmodemPOLL

	// Read Packets
	done := false
	for !done {
		pType := <-rx
		if xmodemDebugFlag {
			fmt.Printf("XMODEM: Packet Type: 0x%x\t", pType)
		}

		switch pType {
		case asciiEOT:
			tx <- asciiACK
			done = true
			if xmodemDebugFlag {
				fmt.Println("Got EOT, done.")
			}
			continue
		case asciiSOH:
			packetSize = xmodemShortPacketLen
		case asciiSTX:
			packetSize = xmodemLongPacketLen
		case asciiCAN:
			return nil, errors.New("XMODEM: Transfer Cancelled by Sender")
		default:
			return nil, errors.New("XMODEM: Protocol Error")
		}

		packetCount := <-rx
		if xmodemDebugFlag {
			fmt.Printf("Block: %d, Size: %d\t", packetCount, packetSize)
		}
		inverseCount := <-rx
		if ^packetCount != inverseCount {
			tx <- asciiNAK
			if xmodemDebugFlag {
				fmt.Println("XMODEM: NAK due to count error")
			}
			continue
		}

		received := 0
		var pData bytes.Buffer
		for received < packetSize {
			pData.WriteByte(<-rx)
			received++
		}

		crc = uint16(<-rx)
		crc <<= 8
		crc |= uint16(<-rx)

		// Calculate CRC
		crcCalc := crc16(pData.Bytes())
		if crcCalc == crc {
			data.Write(pData.Bytes())
			if xmodemDebugFlag {
				fmt.Println("ACK")
			}
			tx <- asciiACK
		} else {
			tx <- asciiNAK
			if xmodemDebugFlag {
				fmt.Println("NAK due to CRC error")
			}
		}
	}
	blob := data.Bytes()
	// remove any trailing EOF indicators (asciiSUBs)
	for blob[len(blob)-1] == asciiSUB {
		blob = blob[:len(blob)-1]
	}
	return blob, nil
}
