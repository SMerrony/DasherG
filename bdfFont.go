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
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-gtk/gdkpixbuf"
)

const (
	maxChars   = 128
	bpp        = 8
	charWidth  = 10
	charHeight = 18 //12
	fontWidth  = 10
	fontHeight = 12
)

type bdfChar struct {
	loaded                           bool
	pixbuf, dimPixbuf, reversePixbuf *gdkpixbuf.Pixbuf
}

var bdfFont [maxChars]bdfChar

func bdfLoad(filename string) {

	fontFile, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Could not open BDF font file <%s>, %v\n", filename, err)
	}
	defer fontFile.Close()
	scanner := bufio.NewScanner(fontFile)
	for scanner.Scan() {
		if strings.TrimRight(scanner.Text(), "\n") == "ENDPROPERTIES" {
			break
		}
	}
	scanner.Scan()
	charCountLine := scanner.Text()
	if !strings.HasPrefix(charCountLine, "CHARS") {
		log.Fatal("CHARS line not found")
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
			log.Fatal("ENCODING line not found")
		}
		asciiCode, _ := strconv.Atoi(encodingLine[9:])
		// skip 2 lines
		scanner.Scan()
		scanner.Scan()
		// decode the BBX line
		scanner.Scan()
		bbxLine := scanner.Text()
		if !strings.HasPrefix(bbxLine, "BBX") {
			log.Fatal("BBX line not found")
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
				}
				lineByte <<= 1
			}
		}
		bdfFont[asciiCode].pixbuf = tmpPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].dimPixbuf = tmpDimPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].reversePixbuf = tmpRevPixbuf.Flip(true).RotateSimple(180).ScaleSimple(charWidth, charHeight, 1)
		bdfFont[asciiCode].loaded = true
	}
	fmt.Printf("bdfFont loaded %d characters\n", charCount)
}
