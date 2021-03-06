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
	"database/sql"
	"math/rand"
	"os"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/atenart/bubbles/beerxml"
)

type DB struct {
	*sql.DB
	Styles  *[]beerxml.Style
	rootdir string
	salt    []byte
}

var structure = []string{
	`
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL,
	password TEXT NOT NULL,
	registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	token TEXT NOT NULL,
	enabled INTEGER DEFAULT 0,
	lang TEXT NOT NULL DEFAULT en,
	--
	CONSTRAINT name UNIQUE (email)
)
`,
	`
CREATE TABLE IF NOT EXISTS recipes (
	id INTEGER PRIMARY KEY,
	user_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	file TEXT NOT NULL,
	public INTEGER DEFAULT 0
)
`,
	`
CREATE TABLE IF NOT EXISTS ingredients (
	id INTEGER PRIMARY KEY,
	user_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	link TEXT DEFAULT "",
	file TEXT NOT NULL,
	--
	CONSTRAINT tuple UNIQUE (user_id, name, type)
)
`,
	`
CREATE TABLE IF NOT EXISTS brews (
	id INTEGER PRIMARY KEY,
	user_id INTEGER_NOT_NULL,
	recipe_id INTEGER NOT NULL,
	step INTEGER DEFAULT 0,
	file TEXT NOT NULL
)
`,
}

// Open a database, and create it if it does not exists.
func Open(rootdir string) (*DB, error) {
	db, err := sql.Open("sqlite3", path.Join(rootdir, "bubbles.db"))
	if err != nil {
		return nil, err
	}

	for _, table := range structure {
		if _, err = db.Exec(table); err != nil {
			// TODO: database may be corrupted.
			db.Close()
			return nil, err
		}
	}

	// Seed the rand source for token generation.
	rand.Seed(time.Now().UnixNano())

	// Retrive all beer styles.
	var xml beerxml.BeerXML
	if err := beerxml.ImportFile(path.Join(rootdir, "styles.xml"), &xml); err != nil {
		return nil, err
	}

	d := &DB{ db, &xml.Styles, rootdir, nil }
	d.salt = d.LoadKey("salt.db", 32)

	return d, nil
}

// Generate a new uniq filename, and create it in $rootdir/
func (db *DB) newUniqFile() (string, error) {
	var file, subdir string
	for {
		token := GenToken(32)
		subdir = token[:2]
		file = token[2:]

		if _, err := os.Stat(path.Join(db.rootdir, subdir, file)); err != nil {
			break
		}
	}

	if err := os.MkdirAll(path.Join(db.rootdir, subdir), 0700); err != nil {
		return "", err
	}

	if _, err := os.Create(path.Join(db.rootdir, subdir, file)); err != nil {
		return "", err
	}

	return path.Join(subdir, file), nil
}

// Token charset.
const charset = "0123456789abcdef"

// Generates a random 32 chars token.
func GenToken(sz int) string {
	token := make([]byte, sz)
	for i := range token {
		token[i] = charset[rand.Intn(len(charset))]
	}
	return string(token)
}

// Import a BeerXML file associated with a DB entry.
func (db *DB) importXML(file string, XML interface{}) error {
	return beerxml.ImportFile(path.Join(db.rootdir, file), XML)
}
