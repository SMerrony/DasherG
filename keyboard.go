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

	"github.com/mattn/go-gtk/gdk"
)

var (
	keyPressEventChan         = make(chan *gdk.EventKey, keyBuffSize)
	keyReleaseEventChan       = make(chan *gdk.EventKey, keyBuffSize)
	ctrlPressed, shiftPressed bool
)

func keyEventHandler() {
	for {
		select {
		case keyPressEvent := <-keyPressEventChan:
			fmt.Println("keyEventHandler got press event")
			switch keyPressEvent.Keyval {
			case gdk.KEY_Control_L, gdk.KEY_Control_R:
				ctrlPressed = true
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock - 1:
				shiftPressed = true
			}

		case keyReleaseEvent := <-keyReleaseEventChan:
			fmt.Println("keyEventHandler got release event")
			switch keyReleaseEvent.Keyval {
			case gdk.KEY_Control_L, gdk.KEY_Control_R:
				ctrlPressed = false
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock - 1:
				shiftPressed = false

			case gdk.KEY_Escape:
				keyboardChan <- '\033'

			case gdk.KEY_Home:
				keyboardChan <- dasherHome
			// case gdk.KEY_Return:
			// 	keyboardChan <- dasherNewLine

			case gdk.KEY_F1:

				// Cursor keys
			case gdk.KEY_Down:
				keyboardChan <- dasherCursorDown
			case gdk.KEY_Left:
				keyboardChan <- dasherCursorLeft
			case gdk.KEY_Right:
				keyboardChan <- dasherCursorRight
			case gdk.KEY_Up:
				keyboardChan <- dasherCursorUp

			default:
				keyByte := byte(keyReleaseEvent.Keyval)
				if ctrlPressed {
					keyByte &= 31 //mask off lower 5 bits
					fmt.Printf("Keystroke modified to <%d>\n", keyByte)
				}
				keyboardChan <- keyByte
			}
		}
	}
}
