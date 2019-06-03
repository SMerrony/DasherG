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

const (
	dasherNul             = 0
	dasherPrintForm       = 1
	dasherRevVideoOff     = 2 // from D210 onwards
	dasherBlinkEnable     = 3 // for the whole screen
	dasherBlinkDisable    = 4 // for the whole screen
	dasherReadWindowAdd   = 5 // requires response...
	dasherAck             = 6 // sent to host to indicatelocal print is complete
	dasherBell            = 7
	dasherHome            = 8 // window home
	dasherTab             = 9
	dasherNewLine         = 10
	dasherEraseEol        = 11
	dasherErasePage       = 12
	dasherCR              = 13
	dasherBlinkOn         = 14
	dasherBlinkOff        = 15
	dasherWriteWindowAddr = 16 // followed by col then row
	dasherPrintScreen     = 17
	dasherRollEnable      = 18
	dasherRollDisable     = 19
	dasherUnderline       = 20
	dasherNormal          = 21
	dasherRevVideoOn      = 22 // from D210 onwards
	dasherCursorUp        = 23
	dasherCursorRight     = 24
	dasherCursorLeft      = 25
	dasherCursorDown      = 26
	dasherDimOn           = 28
	dasherDimOff          = 29
	dasherCmd             = 30

	dasherDelete = 0177 // = 127. or 0x7F
)
