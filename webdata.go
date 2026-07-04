/*
 *    Copyright (c) 2024 Unrud <unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Remote-Touchpad is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU General Public License as published by
 *    the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    Remote-Touchpad is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU General Public License for more details.
 *
 *    You should have received a copy of the GNU General Public License
 *    along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"embed"
	"io/fs"
	"log"
	"mime"
)

//go:embed webdata/*
var webdataFSWithPrefix embed.FS
var webdataFS fs.FS

var webdataTypes = map[string]string{
	".css":  "text/css; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".mjs":  "text/javascript; charset=utf-8",
	".png":  "image/png",
	".woff": "font/woff",
	".json": "application/manifest+json; charset=utf-8",
}

func init() {
	var err error
	webdataFS, err = fs.Sub(webdataFSWithPrefix, "webdata")
	if err != nil {
		log.Fatal(err)
	}
	for ext, typ := range webdataTypes {
		if err := mime.AddExtensionType(ext, typ); err != nil {
			log.Fatal(err)
		}
	}
}
