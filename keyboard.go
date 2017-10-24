package main

import (
	"fmt"

	"github.com/mattn/go-gtk/gdk"
)

var keyEventChan = make(chan *gdk.EventKey)

func keyEventHandler() {
	//ctrlModifier := false
	for {
		keyEvent := <-keyEventChan
		fmt.Println("key-event: ", keyEvent.Keyval)
		fmt.Println("key-ismodifier: ", keyEvent.IsModifier)
		fmt.Println("key-state: ", keyEvent.State)
		// if keyEvent. == gdk.CONTROL_MASK {
		// 	fmt.Println("Caught key press")
		// }
		switch keyEvent.Keyval {

		case gdk.KEY_F1:

		default:
			keyStr := fmt.Sprintf("%c", keyEvent.Keyval)
			keyboardChan <- []byte(keyStr)
		}
	}
}
