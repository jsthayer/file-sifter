// +build !windows
// +build !plan9

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
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// Get extra info about a file: device, nlinks, uid, gid
func statExtended(info os.FileInfo) statEx {
	var xinfo statEx
	sysInf, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		panic("Unsupported type for system file info structure")
	}
	xinfo.device = sysInf.Dev
	xinfo.nlinks = sysInf.Nlink
	xinfo.uid = sysInf.Uid
	xinfo.gid = sysInf.Gid
	xinfo.uidGidValid = true
	return xinfo
}

// Get the current console width
func (self *outputState) getDisplayWidth() int {
	var ws struct {
		rows    uint16
		columns uint16
	}
	syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	width := int(ws.columns)
	if width < 5 || width > 1000 {
		// constrain to sane values
		width = 80
	}
	return width
}

// Register to retrieve window change events
func sigNotifyWindowChange(channel chan os.Signal) {
	signal.Notify(channel, syscall.SIGWINCH)
}
