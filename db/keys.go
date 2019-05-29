// Copyright (C) 2019 Antoine Tenart <antoine.tenart@ack.tf>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package db

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/gorilla/securecookie"
)

// Load a persistent key or generate one if it do not exist.
func (db *DB) LoadKey(name string, length int) []byte {
	path := path.Join(db.rootdir, fmt.Sprintf(".%s.key", name))

	key, err := ioutil.ReadFile(path)
	if err != nil || len(key) != length {
		log.Printf("Generating new key for %s", name)
		return db.genKey(path, length)
	}

	return key
}

// Generate a new random key.
func (db *DB) genKey(path string, length int) []byte {
	key := securecookie.GenerateRandomKey(length)

	if err := ioutil.WriteFile(path, key, 0600); err != nil {
		log.Print(err)
		// Do not fail (the key just won't be persistent).
	}
	return key
}
