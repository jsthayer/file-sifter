/*
	Copyright (C) 2017  John Thayer

	This program is free software; you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation; either version 2 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License along
	with this program; if not, write to the Free Software Foundation, Inc.,
	51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
*/

package sifter

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// The types of messages that may be displayed
const (
	msgNormal = iota // a normal output line
	msgTemp          // a temporary status line on the console
	msgError         // an error message on the console
)

type msgType int

// A message object that gets passed on the output channel to the output thread
type message struct {
	kind msgType // the type of this message
	msg  string  // the line of text to show (should not end with a newline).
}

// Object to manage current output state
type outputState struct {
	stream        io.Writer    // normal output stream (usually stdout or a file)
	errStream     io.Writer    // error/status output stream (usually stderr)
	curTempWidth  int          // length of latest temp status message in runes
	verbosity     int          // current verbosity setting (default=0)
	msgChan       chan message // channel to receive new messages
	exitChan      chan int     // send on this channel when thread exits (at process shutdown)
	displayWidth  int          // current console width in runes
	lineSeparator string       // output line separator (newline unless in --plain0 mode)
}

// Truncate a line of text to fit in the current console width by replacing
// runes in the center with "..." if necessary.
func (self *outputState) truncateMessage(msg string) string {
	runes := []rune(msg)
	width := len(runes)
	halfDispWidth := self.displayWidth/2 - 2
	if width >= self.displayWidth {
		left := runes[:halfDispWidth]
		right := runes[width-halfDispWidth:]
		msg = string(left) + "..." + string(right)
	}
	return msg
}

// Show a temporary status message on the console. (Only for use by the
// output thread.)
func (self *outputState) emitTempf(msg string) {
	// erase any current temp message
	fmt.Fprint(self.errStream, "\r"+strings.Repeat(" ", self.curTempWidth)+"\r")
	// show the new message and save its width
	msg = self.truncateMessage(msg)
	fmt.Fprint(self.errStream, msg)
	self.curTempWidth = utf8.RuneCountInString(msg)
}

// Send a temporary message to the output thread with the given verbosity level.
func (self *outputState) outTempf(verbosity int, format string, a ...interface{}) {
	if self.verbosity >= verbosity {
		self.message(msgTemp, fmt.Sprintf(format, a...))
	}
}

// Send a normal message to the output thread with the given verbosity level.
func (self *outputState) outf(verbosity int, format string, a ...interface{}) {
	if self.verbosity >= verbosity {
		self.message(msgNormal, fmt.Sprintf(format, a...))
	}
}

// Send a message object to the output thread.
func (self *outputState) message(kind msgType, msg string) {
	if self.msgChan != nil {
		self.msgChan <- message{kind, msg}
	}
}

// Goroutine that runs the output thread. Looks for incoming messages,
// window change events (for console resizes), and timer events (to update
// temporary messages at a throttled rate).
func (self *outputState) outputThread() {
	var windowChange = make(chan os.Signal) // receive window change events
	var timer <-chan time.Time              // timer to throttle temp messages
	pending := false                        // a temp message is waiting to be shown after timeout
	pendingMsg := ""                        // the temp message that's pending, if any

	self.displayWidth = self.getDisplayWidth()
	sigNotifyWindowChange(windowChange)

	for {
		select {
		case msg, ok := <-self.msgChan:
			// new message came in
			if !ok {
				// message channel closed due to shutdown; signal main thread we're done
				self.exitChan <- 1
				return
			}
			switch msg.kind {
			case msgNormal:
				// normal message: erase any temp message and print it to output stream
				self.emitTempf("")
				fmt.Fprint(self.stream, msg.msg)
				fmt.Fprint(self.stream, self.lineSeparator)
				pending = false
			case msgError:
				// error message: erase any temp message and print it to console
				self.emitTempf("")
				fmt.Fprintln(self.errStream, msg.msg)
				pending = false
			case msgTemp:
				// temp message
				if timer == nil {
					// no throttle; show on console and start a throttle timer
					self.emitTempf(msg.msg)
					timer = time.After(time.Millisecond * 100)
				} else {
					// throttle timer active; just save pending message
					pending = true
					pendingMsg = msg.msg
				}
			}
		case <-timer:
			// throttle timer expired; show any pending message
			timer = nil
			if pending {
				self.emitTempf(pendingMsg)
				pending = false
			}
		case <-windowChange:
			// window size changed; update display width value
			self.displayWidth = self.getDisplayWidth()
		}
	}
}
