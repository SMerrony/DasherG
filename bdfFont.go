// Copyright (C) 2017,2019 Steve Merrony

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
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mattn/go-gtk/gdkpixbuf"
)

const (
	maxChars = 128
	bpp      = 8
	// width (pixels) of a char in the raw font
	fontWidth = 10
	// height (pixels) of a char in the raw font
	fontHeight = 12
)

type bdfChar struct {
	loaded                           bool
	pixbuf, dimPixbuf, reversePixbuf *gdkpixbuf.Pixbuf
	pixels                           [fontWidth][fontHeight]bool
}

var (
	bdfFont [maxChars]bdfChar
	// charWidth is the currently displayed width of a character
	charWidth int
	// charHeight is the currently displayed height of a character
	charHeight int
)

func bdfLoad(filename string, zoom int) {
	switch zoom {
	case zoomLarge:
		charWidth, charHeight = 10, 24
	case zoomNormal:
		charWidth, charHeight = 10, 18
	case zoomSmaller:
		charWidth, charHeight = 8, 12
	case zoomTiny:
		charWidth, charHeight = 7, 10
	}

	fontData, err := Asset(filename)
	if err != nil {
		log.Fatalf("Could not load BDF font resource<%s>, %v\n", filename, err)
	}

	buffer := bytes.NewBuffer(fontData)
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		if strings.TrimRight(scanner.Text(), "\n") == "ENDPROPERTIES" {
			break
		}
	}
	scanner.Scan()
	charCountLine := scanner.Text()
	if !strings.HasPrefix(charCountLine, "CHARS") {
		log.Fatal("bdfFont: CHARS line not found")
	}
	charCount, _ := strconv.Atoi(charCountLine[6:])

	for cc := 0; cc < charCount; cc++ {
		tmpPixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, fontWidth, fontHeight)
		tmpDimPixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, fontWidth, fontHeight)
		tmpRevPixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, fontWidth, fontHeight)

		for !strings.HasPrefix(scanner.Text(), "STARTCHAR") {
			scanner.Scan()
		}
		scanner.Scan()
		encodingLine := scanner.Text()
		if !strings.HasPrefix(encodingLine, "ENCODING") {
			log.Fatal("bdfFont: ENCODING line not found")
		}
		asciiCode, _ := strconv.Atoi(encodingLine[9:])
		// skip 2 lines
		scanner.Scan()
		scanner.Scan()
		// decode the BBX line
		scanner.Scan()
		bbxLine := scanner.Text()
		if !strings.HasPrefix(bbxLine, "BBX") {
			log.Fatal("bdfFont: BBX line not found")
		}
		bbxTokens := strings.Split(scanner.Text(), " ")
		pixWidth, _ := strconv.Atoi(bbxTokens[1])
		pixHeight, _ := strconv.Atoi(bbxTokens[2])
		xOffset, _ := strconv.Atoi(bbxTokens[3])
		yOffset, _ := strconv.Atoi(bbxTokens[4])
		// skip the BITMAP line
		scanner.Scan()
		// load the actual bitmap for this char a row at a time from the top down
		tmpPixbuf.Fill(0)
		tmpDimPixbuf.Fill(0)
		tmpRevPixbuf.Fill(255 << 16)
		for bitMapLine := pixHeight - 1; bitMapLine >= 0; bitMapLine-- {
			scanner.Scan()
			lineStr := scanner.Text()
			lineByte, _ := strconv.ParseUint(lineStr, 16, 16)
			for i := 0; i < pixWidth; i++ {
				pix := ((lineByte & 0x80) >> 7) == 1 // test the MSB
				if pix {
					nChannels := tmpPixbuf.GetNChannels()
					rowStride := tmpPixbuf.GetRowstride()
					tmpPixbuf.GetPixels()[((yOffset+bitMapLine)*rowStride)+((xOffset+i)*nChannels)+1] = 255
					tmpDimPixbuf.GetPixels()[((yOffset+bitMapLine)*rowStride)+((xOffset+i)*nChannels)+1] = 128
					tmpRevPixbuf.GetPixels()[((yOffset+bitMapLine)*rowStride)+((xOffset+i)*nChannels)+1] = 0
					bdfFont[asciiCode].pixels[xOffset+i][yOffset+bitMapLine] = true
				}
				lineByte <<= 1
			}
		}
		bdfFont[asciiCode].pixbuf = tmpPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].dimPixbuf = tmpDimPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].reversePixbuf = tmpRevPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].loaded = true
	}
	fmt.Printf("INFO: bdfFont loaded %d DASHER characters\n", charCount)
}
