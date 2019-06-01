// This is based on "chriszzzzz"'s fork of Omegaice's go-xmodem code

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
//

package main

import (
	"bytes"
	"errors"
	"io"
	"log"
)

const SOH byte = 0x01
const STX byte = 0x02
const EOT byte = 0x04
const ACK byte = 0x06
const NAK byte = 0x15
const CAN byte = 0x18
const POLL byte = 'C'

const SHORT_PACKET_PAYLOAD_LEN = 128
const LONG_PACKET_PAYLOAD_LEN = 1024

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
	startByte := SOH
	if packetPayloadLen == LONG_PACKET_PAYLOAD_LEN {
		startByte = STX
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
		toSend.Write([]byte{EOT})
	}

	sent := 0
	for sent < toSend.Len() {
		if n, err := c.Write(toSend.Bytes()[sent:]); err != nil {
			return err
		} else {
			sent += n
		}
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
	return xmodemSend(c, data, SHORT_PACKET_PAYLOAD_LEN)
}

func XModemSend1K(c io.ReadWriter, data []byte) error {
	return xmodemSend(c, data, LONG_PACKET_PAYLOAD_LEN)
}

func xmodemSend(c io.ReadWriter, data []byte, packetPayloadLen int) error {
	oBuffer := make([]byte, 1)

	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] == POLL {
		var blocks int = len(data) / packetPayloadLen
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

			if oBuffer[0] == ACK {
				currentBlock++
			} else {
				failed++
			}
		}

		if _, err := c.Write([]byte{EOT}); err != nil {
			return err
		}
	}

	return nil
}

func XModemReceive(rx chan byte, tx chan byte) ([]byte, error) {
	var data bytes.Buffer
	// oBuffer := make([]byte, 1)
	// dBuffer := make([]byte, LONG_PACKET_PAYLOAD_LEN)

	log.Println("Before")

	// Start Connection
	// if _, err := c.Write([]byte{POLL}); err != nil {
	// 	return nil, err
	// }
	tx <- POLL

	log.Println("Write Poll")

	// Read Packets
	done := false
	for !done {
		// if _, err := c.Read(oBuffer); err != nil {
		// 	return nil, err
		// }
		//pType := oBuffer[0]
		pType := <-rx
		log.Printf("PType: 0x%x\n", pType)

		// if pType == EOT {
		// 	if _, err := c.Write([]byte{ACK}); err != nil {
		// 		return nil, err
		// 	}
		// 	break
		// }

		var packetSize int
		switch pType {
		case EOT:
			// if _, err := c.Write([]byte{ACK}); err != nil {
			// 	return nil, err
			// }
			tx <- ACK
			done = true
			continue
			// break
		case SOH:
			packetSize = SHORT_PACKET_PAYLOAD_LEN
			// break
		case STX:
			packetSize = LONG_PACKET_PAYLOAD_LEN
			// break
		case CAN:
			return nil, errors.New("Cancelled")
		default:
			return nil, errors.New("XMODEM Protocol Error")
		}

		// if _, err := c.Read(oBuffer); err != nil {
		// 	return nil, err
		// }
		// packetCount := oBuffer[0]
		// dBuffer = <-rx
		packetCount := <-rx

		// if _, err := c.Read(oBuffer); err != nil {
		// 	return nil, err
		// }
		// inverseCount := oBuffer[0]
		inverseCount := <-rx

		if packetCount > inverseCount || inverseCount+packetCount != 255 {
			// if _, err := c.Write([]byte{NAK}); err != nil {
			// 	return nil, err
			// }
			tx <- NAK
			continue
		}

		received := 0
		var pData bytes.Buffer
		for received < packetSize {
			// n, err := c.Read(dBuffer)
			// if err != nil {
			// 	return nil, err
			// }
			pData.WriteByte(<-rx)
			// received += n
			received++
			// pData.Write(dBuffer[:n])
		}

		var crc uint16
		// if _, err := c.Read(oBuffer); err != nil {
		// 	return nil, err
		// }
		// crc = uint16(oBuffer[0])
		// dBuffer = <-rx
		crc = uint16(<-rx)

		// if _, err := c.Read(oBuffer); err != nil {
		// 	return nil, err
		// }
		crc <<= 8
		// crc |= uint16(oBuffer[0])
		// dBuffer = <-rx
		crc |= uint16(<-rx)

		// Calculate CRC
		crcCalc := crc16(pData.Bytes())
		if crcCalc == crc {
			data.Write(pData.Bytes())
			// if _, err := c.Write([]byte{ACK}); err != nil {
			// 	return nil, err
			// }
			tx <- ACK
		} else {
			// if _, err := c.Write([]byte{NAK}); err != nil {
			// 	return nil, err
			// }
			tx <- NAK
		}
	}

	return data.Bytes(), nil
}
