# DasherG
DasherG is a free terminal emulator for Data General DASHER series character-based terminals.  It is written in [Go](https://golang.org/) using the [Fyne](https://fyne.io) toolkit and should run on all common platforms supported by Go.

![screenshot](screenshots/DasherG_v0_16_0.png "DasherG Screenshot")

## Key Features

* DASHER D200 & D210 Emulation
* Serial interface support at 300, 1200, 2400, 4800, 9600 & 19200 baud, 7 or 8 data bits (defaults to 9600, 8, n, 1)
* BREAK key support for serial interface - permits use as master console
* Network Interface (Telnet) support
* May specify ```-host=host:port``` on command line
* Reverse video, blinking, dim and underlined characters
* Pixel-for-pixel copy of D410 character set
* 15 (plus Ctrl & Shift) DASHER Function keys, Erase Page, Erase EOL, Hold, Local Print and Break keys
* C1, C2, C3 and C4 DASHER Custom keys
* Loadable function key templates (BROWSE, SED and SMI provided as examples)
* 1000-line terminal history
* Session logging to file
* Various terminal widths, heights and zoom-levels available
* Support for mini-Expect scripts to automate some tasks [see Wiki](https://github.com/SMerrony/DasherG/wiki/DasherG-Mini-Expect-Scripts)
* Copy and Paste - select region with mouse (it is automatically copied to clipboard) and paste at cursor via Edit menu
* XMODEM-CRC file transfer with short (128) or long (1024) packets

## Download
DasherG is [hosted on GitHub](https://github.com/SMerrony/DasherG).

## Build from Source
### Prerequisites
You will need a working, recent (v1.24 or later) Go compiler.  The first time you build or install
DasherG it might pull in the toolkit dependencies - this will take a little while.  Subsequent
builds should be very quick.

### Build
```
cd cmd/DasherG
go build
```

or, if you prefer ```go install```

## Running DasherG
From the build or install directory simply type

```./DasherG```

Optionally, you may add the ```-host=host:port``` argument to connect to a running host via telnet. Eg. 

```./DasherG -host=localhost:23```

For a full list of all available DasherG options type

```./DasherG -h```

### Function and Special Keys
You may have to use the keys simulated on the toolbar in DasherG as your OS might interfere with the physical function keys on your keyboard.  The Shift and Control keys can be used in conjunction with the simulated F-keys just like a real Dasher.

The "Brk" button sends a Command-Break signal to the host when connected via the serial interface.

"Hold" and "Local Print" work as you would expect, although the print actually goes to a user-specified image (PNG) file.

### Bell Sound

For the system bell to operate, DasherG must have been started from a terminal which supports the bell.

### Emulation Details
[See here](https://github.com/SMerrony/DasherG/blob/master/implementationChart.md)
