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

import "os"

// Get extra info about a file: device, nlinks, uid, gid (not supported in Windows)
func statExtended(info os.FileInfo) statEx {
	var xinfo statEx
	xinfo.nlinks = 1
	return xinfo
}

// TODO: figure out how to get actual console width and notifications
func (self *outputState) getDisplayWidth() int {
	return 80
}

func sigNotifyWindowChange(channel chan os.Signal) {
}
