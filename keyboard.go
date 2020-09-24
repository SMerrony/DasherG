// Copyright ©2017,2018,2020 Steve Merrony

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

	"fyne.io/fyne"
	"github.com/mattn/go-gtk/gdk"
)

var (
	keyPressEventChan         = make(chan *gdk.EventKey, keyBuffSize)
	keyReleaseEventChan       = make(chan *gdk.EventKey, keyBuffSize)
	keyDownEventChan          = make(chan *fyne.KeyEvent, keyBuffSize)
	keyUpEventChan            = make(chan *fyne.KeyEvent, keyBuffSize)
	ctrlPressed, shiftPressed bool
)

func keyEventHandler(kbdChan chan<- byte) {
	for {
		select {
		case keyPressEvent := <-keyPressEventChan:
			//fmt.Println("keyEventHandler got press event")
			switch keyPressEvent.Keyval {
			case gdk.KEY_Control_L, gdk.KEY_Control_R:
				ctrlPressed = true
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock - 1:
				shiftPressed = true
			}

		case keyReleaseEvent := <-keyReleaseEventChan:
			//fmt.Println("keyEventHandler got release event")
			switch keyReleaseEvent.Keyval {
			case gdk.KEY_Control_L, gdk.KEY_Control_R:
				ctrlPressed = false
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock - 1:
				shiftPressed = false

			case gdk.KEY_Escape:
				kbdChan <- '\033'

			case gdk.KEY_Home:
				kbdChan <- dasherHome

			case gdk.KEY_Delete: // the DEL key must map to 127 which is the DASHER DEL code
				kbdChan <- modify(127)

			case gdk.KEY_F1:
				kbdChan <- dasherCmd
				kbdChan <- modify(113)
			case gdk.KEY_F2:
				kbdChan <- dasherCmd
				kbdChan <- modify(114)
			case gdk.KEY_F3:
				kbdChan <- dasherCmd
				kbdChan <- modify(115)
			case gdk.KEY_F4:
				kbdChan <- dasherCmd
				kbdChan <- modify(116)
			case gdk.KEY_F5:
				kbdChan <- dasherCmd
				kbdChan <- modify(117)

			case gdk.KEY_F6:
				kbdChan <- dasherCmd
				kbdChan <- modify(118)
			case gdk.KEY_F7:
				kbdChan <- dasherCmd
				kbdChan <- modify(119)
			case gdk.KEY_F8:
				kbdChan <- dasherCmd
				kbdChan <- modify(120)
			case gdk.KEY_F9:
				kbdChan <- dasherCmd
				kbdChan <- modify(121)
			case gdk.KEY_F10:
				kbdChan <- dasherCmd
				kbdChan <- modify(122)

			case gdk.KEY_F11:
				kbdChan <- dasherCmd
				kbdChan <- modify(123)
			case gdk.KEY_F12:
				kbdChan <- dasherCmd
				kbdChan <- modify(124)
			case gdk.KEY_F13:
				kbdChan <- dasherCmd
				kbdChan <- modify(125)
			case gdk.KEY_F14:
				kbdChan <- dasherCmd
				kbdChan <- modify(126)
			case gdk.KEY_F15:
				kbdChan <- dasherCmd
				kbdChan <- modify(112)

				// Cursor keys
			case gdk.KEY_Down:
				kbdChan <- dasherCursorDown
			case gdk.KEY_Left:
				kbdChan <- dasherCursorLeft
			case gdk.KEY_Right:
				kbdChan <- dasherCursorRight
			case gdk.KEY_Up:
				kbdChan <- dasherCursorUp

			default:
				keyByte := byte(keyReleaseEvent.Keyval)
				if ctrlPressed {
					keyByte &= 31 //mask off lower 5 bits
					//fmt.Printf("Keystroke modified to <%d>\n", keyByte)
				}
				kbdChan <- keyByte
			}
		}
	}
}

func keyEventHandler2(kbdChan chan<- byte) {
	for {
		select {
		case keyPressEvent := <-keyDownEventChan:
			fmt.Println("keyEventHandler got press event")
			switch keyPressEvent.Name {
			case "LeftControl", "RightControl":
				ctrlPressed = true
			case "LeftShift", "RightShift", "CapsLock":
				shiftPressed = true
			}

		case keyReleaseEvent := <-keyUpEventChan:
			fmt.Printf("keyEventHandler got release event for <%s>\n", keyReleaseEvent.Name)
			switch keyReleaseEvent.Name {
			case "LeftControl", "RightControl":
				ctrlPressed = false
			case "LeftShift", "RightShift", "CapsLock":
				shiftPressed = false

			case "Escape":
				kbdChan <- '\033'

			case "Home":
				kbdChan <- dasherHome

			case "Delete": // the DEL key must map to 127 which is the DASHER DEL code
				kbdChan <- modify(127)

			case "F1":
				kbdChan <- dasherCmd
				kbdChan <- modify(113)
			case "F2":
				kbdChan <- dasherCmd
				kbdChan <- modify(114)
			case "F3":
				kbdChan <- dasherCmd
				kbdChan <- modify(115)
			case "F4":
				kbdChan <- dasherCmd
				kbdChan <- modify(116)
			case "F5":
				kbdChan <- dasherCmd
				kbdChan <- modify(117)

			case "F6":
				kbdChan <- dasherCmd
				kbdChan <- modify(118)
			case "F7":
				kbdChan <- dasherCmd
				kbdChan <- modify(119)
			case "F8":
				kbdChan <- dasherCmd
				kbdChan <- modify(120)
			case "F9":
				kbdChan <- dasherCmd
				kbdChan <- modify(121)
			case "F10":
				kbdChan <- dasherCmd
				kbdChan <- modify(122)

			case "F11":
				kbdChan <- dasherCmd
				kbdChan <- modify(123)
			case "F12":
				kbdChan <- dasherCmd
				kbdChan <- modify(124)
			case "F13":
				kbdChan <- dasherCmd
				kbdChan <- modify(125)
			case "F14":
				kbdChan <- dasherCmd
				kbdChan <- modify(126)
			case "F15":
				kbdChan <- dasherCmd
				kbdChan <- modify(112)

				// Cursor keys
			case "Down":
				kbdChan <- dasherCursorDown
			case "Left":
				kbdChan <- dasherCursorLeft
			case "Right":
				kbdChan <- dasherCursorRight
			case "Up":
				kbdChan <- dasherCursorUp

			default:
				keyByte := byte(keyReleaseEvent.Name[0])
				if ctrlPressed {
					keyByte &= 31 //mask off lower 5 bits
					//fmt.Printf("Keystroke modified to <%d>\n", keyByte)
				}
				kbdChan <- keyByte
			}
		}
	}
}

func modify(k byte) byte {
	var modifier byte
	if shiftPressed {
		modifier -= 16
	}
	if ctrlPressed {
		modifier -= 64
	}
	return k + modifier
}
