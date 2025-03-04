# Reminders for developing DasherG

## GoRoutines
* localListener
* terminal.updateListener
* keyEventHandler
* terminal.run
* drawCrt (via call in dasherg.go main)
* updateStatusBox (via call in  dasherg.go buildStatusBox)
  
* blink timer (via call in dasherg.go setupWindow)

* expectRunner
* serialReader
* serialWriter
* telnetReader
* telnetWriter

## Files containing any GUI toolkit references
* crt.go
* dasherg.go
* fKeyMatrix.go
* keyboard.go
* menuHandlers.go
* miniExpect.go

## Terminal setting on Linux host:

Ensure `ncurses-term` and `inetutils-telnetd` packages are installed.

`export TERM=d210-dg`

You may have to `stty echo` if no characters appear when you type.

N.B. There are bugs in the termcap database for (all) the DASHER terminals; not many allegedly termcap (ncurses) aware programs actually handle unusual terminal types.  The `htop` program does a fair job of behaving properly - even so, you will see a few glitches over time. Likewise with `iftop`.  The  `nano` editor seems to behave quite well - although you may need to use the function keys to save and exit.

## Serial Port Test Setup (Linux)

Install and then `insmod` `tty0tty`.

Then `/dev/tnt0` is connected to `/dev/tnt1`, etc.
