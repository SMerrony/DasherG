package main

const (
	telnetCmdSE   = 240
	telnetCmdNOP  = 241
	telnetCmdDM   = 242
	telnetCmdBRK  = 243
	telnetCmdIP   = 244
	telnetCmdAO   = 245
	telnetCmdAYT  = 246
	telnetCmdEC   = 247
	telnetCmdEL   = 248
	telnetCmdGA   = 249
	telnetCmdSB   = 250
	telnetCmdWILL = 251
	telnetCmdWONT = 252
	telnetCmdDO   = 253
	telnetCmdDONT = 254
	telnetCmdIAC  = 255

	telnetOptBIN    = 0
	telnetOptECHO   = 1
	telnetOptRECON  = 2
	telnetOptSGA    = 3
	telnetOptSTATUS = 5
	telnetOptCOLS   = 8
	telnetOptROWS   = 9
	telnetOptEASCII = 17
	telnetOptLOGOUT = 18
	telnetOptTTYPE  = 24
	telnetOptNAWS   = 31 // window size
	telnetOptTSPEED = 32
	telnetOptXDISP  = 35
	telnetOptNEWENV = 39
)
