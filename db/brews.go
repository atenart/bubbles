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
	"os"
	"path"

	"github.com/atenart/bubbles/beerxml"
)

// Retrieve a single brew given its id.
func (db *DB) GetBrew(id int64) (*Brew, error) {
	var b Brew
	err := db.QueryRow("SELECT * FROM brews WHERE id == $1", id).
	       Scan(&b.Id, &b.UserId, &b.RecipeId, &b.Step, &b.File)
	if err != nil {
		return nil, err
	}

	if err := db.importXML(b.File, &b.XML); err != nil {
		return nil, err
	}

	return &b, nil
}

// Retrieve all brews for a given user.
func (db *DB) GetUserBrews(uid int64) ([]*Brew, error) {
	row, err := db.Query("SELECT * FROM brews WHERE user_id == ?", uid)
	if err != nil {
		return nil, err
	}

	var brews []*Brew
	for row.Next() {
		var b Brew
		row.Scan(&b.Id, &b.UserId, &b.RecipeId, &b.Step, &b.File)

		brews = append(brews, &b)
		if err := db.importXML(b.File, &b.XML); err != nil {
			return nil, err
		}
	}

	return brews, nil
}

// Add a new brew.
func (db *DB) AddBrew(b *Brew) (int64, error) {
	var err error
	if b.File, err = db.newUniqFile(); err != nil {
		return -1, err
	}

	result, err := db.Exec(`
INSERT INTO brews (user_id, recipe_id, step, file)
VALUES (?, ?, ?, ?)`, b.UserId, b.RecipeId, b.Step, b.File)
	if err != nil {
		os.Remove(b.File)
		return -1, err
	}

	// Get the generated id in response to the previous command.
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	if err := beerxml.ExportFile(b.XML, path.Join(db.rootdir, b.File)); err != nil {
		return -1, err
	}

	return id, nil
}

// Update a brew.
func (db *DB) UpdateBrew(b *Brew) error {
	_, err := db.Exec(`
REPLACE INTO brews (id, user_id, recipe_id, step, file)
VALUES (?, ?, ?, ?, ?)`, b.Id, b.UserId, b.RecipeId, b.Step, b.File)
	if err != nil {
		return err
	}

	return beerxml.ExportFile(b.XML, path.Join(db.rootdir, b.File))
}
