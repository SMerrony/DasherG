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

	dasherDelete = 0177
)
