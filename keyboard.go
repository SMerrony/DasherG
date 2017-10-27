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
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock:
				shiftPressed = true
			}

		case keyReleaseEvent := <-keyReleaseEventChan:
			fmt.Println("keyEventHandler got release event")
			switch keyReleaseEvent.Keyval {
			case gdk.KEY_Control_L, gdk.KEY_Control_R:
				ctrlPressed = false
			case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Shift_Lock:
				shiftPressed = false

			case gdk.KEY_Escape:
				keyboardChan <- '\036'

			case gdk.KEY_Home:
				keyboardChan <- dasherHome
			case gdk.KEY_Return:
				keyboardChan <- dasherNewLine

			case gdk.KEY_F1:

				// Cursor keys
			case gdk.KEY_Down:
				keyboardChan <- '\032'
			case gdk.KEY_Left:
				keyboardChan <- '\031'
			case gdk.KEY_Right:
				keyboardChan <- '\030'
			case gdk.KEY_Up:
				keyboardChan <- '\027'

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
