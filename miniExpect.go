// dasherg.go

// Copyright (C) 2018,2021  Steve Merrony

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

// This file implements the mini-Expect automated scripting "language" for DasherG.
// Mini-Expect is documented in the DasherG Wiki on GitHub.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-gtk/gtk"
)

// expectRunner must be run as a Goroutine - not in the main loop
func expectRunner(expectFile *os.File, expectChan <-chan byte, kbdChan chan<- byte, term *terminalT) {
	defer expectFile.Close()
	term.rwMutex.Lock()
	term.expecting = true
	term.rwMutex.Unlock()
	scanner := bufio.NewScanner(expectFile)
scriptLoop:
	for scanner.Scan() {
		expectLine := scanner.Text()
		if len(expectLine) == 0 {
			continue
		}
		if traceExpect {
			fmt.Printf("DEBUG: Expect line <%s>\n", expectLine)
		}
		if expectLine[:1] == "#" {
			if traceExpect {
				fmt.Printf("DEBUG: Ignoring comment line <%s>\n", expectLine)
			}
			continue
		}
		switch {
		// expect
		case strings.HasPrefix(expectLine, "expect"):
			expectStr := strings.Split(expectLine, "\"")[1]
			hostString := ""
			found := false
			time.Sleep(200 * time.Millisecond)
			for {
				select {
				case b := <-expectChan:
					if b == dasherCR || b == dasherNewLine {
						hostString = ""
					} else {
						hostString += string(b)
						if traceExpect {
							fmt.Printf("DEBUG: Expect want <%s>, response so far is: <%s>\n", expectStr, hostString)
						}
						if strings.HasSuffix(hostString, expectStr) {
							found = true
							break
						}
					}
				}
				if found {
					if traceExpect {
						fmt.Printf("DEBUG: found expect string<%s>\n", expectStr)
					}
					time.Sleep(100 * time.Millisecond)
					break
				}
			}

		// send
		case strings.HasPrefix(expectLine, "send"):
			sendLine := strings.Split(expectLine, "\"")[1]
			if traceExpect {
				fmt.Printf("DEBUG: send line <%s>\n", sendLine)
			}
			sendLine = strings.Replace(sendLine, "\\n", fmt.Sprintf("%c", 0x0D), -1)
			for _, ch := range sendLine {
				if traceExpect {
					fmt.Printf("DEBUG: sending char <%c>\n", ch)
				}
				kbdChan <- byte(ch)
				// the following delay seems to be crucial...
				time.Sleep(150 * time.Millisecond)
			}

		// exit
		case strings.HasPrefix(expectLine, "exit"):
			if traceExpect {
				fmt.Println("DEBUG: exiting mini-Expect")
			}
			break scriptLoop

		default:
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Unknown command in mini-Expect script file")
			ed.Run()
			ed.Destroy()
			break scriptLoop
		}
	}
	term.rwMutex.Lock()
	term.expecting = false
	term.rwMutex.Unlock()
}
