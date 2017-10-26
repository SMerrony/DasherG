package main

import (
	"fmt"

	"github.com/mattn/go-gtk/gdk"
)

var (
	keyPressEventChan   = make(chan *gdk.EventKey, keyBuffSize)
	keyReleaseEventChan = make(chan *gdk.EventKey, keyBuffSize)
)

func keyEventHandler() {
	for {
		select {
		case _ = <-keyPressEventChan:
			fmt.Println("keyEventHandler got press event")

		case keyReleaseEvent := <-keyReleaseEventChan:
			fmt.Println("keyEventHandler got release event")
			switch keyReleaseEvent.Keyval {

			case gdk.KEY_Escape:
				keyboardChan <- byte('\036')

			case gdk.KEY_F1:

			default:
				keyboardChan <- byte(keyReleaseEvent.Keyval)
			}
		}
	}
}
