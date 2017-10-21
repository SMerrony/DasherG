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
	charHeight = 12
)

type bdfChar struct {
	loaded bool
	//pixmap, dimPixmap, reversePixmap gdk.Pixmap
	pixbuf *gdkpixbuf.Pixbuf
}

var bdfFont [maxChars]bdfChar

func bdfLoad(filename string) {
	// for c := range bdfFont {
	// 	bdfFont[c].loaded = false
	// 	//bdfFont[c].pixmap = gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, charWidth, charHeight)
	// 	//bdfFont[c].dimPixmap = gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, charWidth, charHeight)
	// 	//bdfFont[c].reversePixmap = gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, charWidth, charHeight)
	// }

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
		tmpPixbuf := gdkpixbuf.NewPixbuf(gdkpixbuf.GDK_COLORSPACE_RGB, false, bpp, charWidth, charHeight)
		//var tmpPixmap gdk.Pixmap

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
		for bitMapLine := pixHeight - 1; bitMapLine >= 0; bitMapLine-- {
			scanner.Scan()
			lineStr := scanner.Text()
			lineByte, _ := strconv.ParseUint(lineStr, 16, 16)
			for i := 0; i < pixWidth; i++ {
				pix := ((lineByte & 0x80) >> 7) == 1 // test the MSB
				if pix {
					nChannels := tmpPixbuf.GetNChannels()
					rowStride := tmpPixbuf.GetRowstride()
					pixels := tmpPixbuf.GetPixels()
					pixels[(yOffset*rowStride)+(xOffset*nChannels)] = 255
				}
				lineByte <<= 1
			}
		}
		bdfFont[asciiCode].pixbuf = tmpPixbuf
		bdfFont[asciiCode].loaded = true
	}
	fmt.Printf("bdfFont loaded %d characters\n", charCount)
}
