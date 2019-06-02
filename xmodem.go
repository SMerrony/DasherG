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
	"io"
	"log"
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

func sendBlock(c io.ReadWriter, block int, data []byte, packetPayloadLen int) error {
	startByte := asciiSOH
	if packetPayloadLen == xmodemLongPacketLen {
		startByte = asciiSTX
	}
	// send start byte and length
	var hdr []byte
	hdr = append(hdr, startByte, byte(uint8(block%256)), byte(uint8(255-(block%256))))
	if _, err := c.Write(hdr); err != nil {
		return err
	}
	// if _, err := c.Write([]byte{startByte}); err != nil {
	// 	return err
	// }
	// if _, err := c.Write([]byte{uint8(block % 256)}); err != nil {
	// 	return err
	// }
	// if _, err := c.Write([]byte{uint8(255 - (block % 256))}); err != nil {
	// 	return err
	// }

	//send data
	var toSend bytes.Buffer
	toSend.Write(data)
	for toSend.Len() < packetPayloadLen {
		toSend.Write([]byte{asciiEOT})
	}

	sent := 0
	for sent < toSend.Len() {
		n, err := c.Write(toSend.Bytes()[sent:])
		if err != nil {
			return err
		}
		sent += n
	}

	//calc CRC
	u16CRC := crc16Constant(data, packetPayloadLen)

	//send CRC
	if _, err := c.Write([]byte{uint8(u16CRC >> 8)}); err != nil {
		return err
	}
	if _, err := c.Write([]byte{uint8(u16CRC & 0x0FF)}); err != nil {
		return err
	}

	return nil
}

func XModemSend(c io.ReadWriter, data []byte) error {
	return xmodemSend(c, data, xmodemShortPacketLen)
}

func XModemSend1K(c io.ReadWriter, data []byte) error {
	return xmodemSend(c, data, xmodemLongPacketLen)
}

func xmodemSend(c io.ReadWriter, data []byte, packetPayloadLen int) error {
	oBuffer := make([]byte, 1)

	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] == xmodemPOLL {
		var blocks = len(data) / packetPayloadLen
		if len(data) > blocks*packetPayloadLen {
			blocks++
		}

		failed := 0
		var currentBlock = 0
		for currentBlock < blocks && failed < 10 {
			if int(int(currentBlock+1)*int(packetPayloadLen)) > len(data) {
				sendBlock(c, currentBlock+1, data[currentBlock*packetPayloadLen:], packetPayloadLen)
			} else {
				sendBlock(c, currentBlock+1, data[currentBlock*packetPayloadLen:(currentBlock+1)*packetPayloadLen], packetPayloadLen)
			}

			if _, err := c.Read(oBuffer); err != nil {
				return err
			}

			if oBuffer[0] == asciiACK {
				currentBlock++
			} else {
				failed++
			}
		}

		if _, err := c.Write([]byte{asciiEOT}); err != nil {
			return err
		}
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
	log.Println("Before")

	// Start Connection
	tx <- xmodemPOLL

	// Read Packets
	done := false
	for !done {
		pType := <-rx
		log.Printf("PType: 0x%x\n", pType)

		switch pType {
		case asciiEOT:
			tx <- asciiACK
			done = true
			continue
		case asciiSOH:
			packetSize = xmodemShortPacketLen
		case asciiSTX:
			packetSize = xmodemLongPacketLen
		case asciiCAN:
			return nil, errors.New("XMODEM Transfer asciiCancelled by Sender")
		default:
			return nil, errors.New("XMODEM Protocol Error")
		}

		packetCount := <-rx
		inverseCount := <-rx
		if packetCount > inverseCount || inverseCount+packetCount != 255 {
			tx <- asciiNAK
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
			tx <- asciiACK
		} else {
			tx <- asciiNAK
		}
	}
	blob := data.Bytes()
	// remove any trailing EOF indicators (asciiSUBs)
	for blob[len(blob)-1] == asciiSUB {
		blob = blob[:len(blob)-1]
	}
	return blob, nil
}
