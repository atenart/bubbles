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
	_ "github.com/mattn/go-sqlite3"
)

// Retrives a recipe given its id.
func (db *DB) GetRecipe(id int64) (*Recipe, error) {
	var r Recipe
	err := db.QueryRow("SELECT * FROM recipes WHERE id == $1", id).
		Scan(&r.Id, &r.UserId, &r.Name, &r.File, &r.Public)
	if err != nil {
		return nil, err
	}

	if err := db.importXML(r.File, &r.XML); err != nil {
		return nil, err
	}

	return &r, nil
}

// Retrieves all recipes for a given user.
func (db *DB) GetUserRecipes(uid int64) ([]*Recipe, error) {
	row, err := db.Query("SELECT * FROM recipes WHERE user_id == ? ORDER BY id DESC", uid)
	if err != nil {
		return nil, err
	}

	var recipes []*Recipe
	for row.Next() {
		var r Recipe
		row.Scan(&r.Id, &r.UserId, &r.Name, &r.File, &r.Public)

		recipes = append(recipes, &r)
		if err := db.importXML(r.File, &r.XML); err != nil {
			return nil, err
		}
	}

	return recipes, nil
}

// Add a new recipe.
func (db *DB) AddRecipe(r *Recipe) (int64, error) {
	var err error
	if r.File, err = db.newUniqFile(); err != nil {
		return -1, err
	}

	result, err := db.Exec(`
INSERT INTO recipes (user_id, name, file, public)
VALUES (?, ?, ?, ?)`, r.UserId, r.Name, r.File, r.Public)
	if err != nil {
		os.Remove(r.File)
		return -1, err
	}

	// Get the generated id in response to the previous command.
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	if err := beerxml.ExportFile(r.XML, path.Join(db.rootdir, r.File)); err != nil {
		return -1, err
	}

	return id, nil
}

// Update a recipe.
func (db *DB) UpdateRecipe(r *Recipe) error {
	_, err := db.Exec(`
REPLACE INTO recipes (id, user_id, name, file, public)
VALUES (?, ?, ?, ?, ?)`, r.Id, r.UserId, r.Name, r.File, r.Public)
	if err != nil {
		return err
	}

	return beerxml.ExportFile(&r.XML, path.Join(db.rootdir, r.File))
}

// Delete a recipe.
func (db *DB) DeleteRecipe(r *Recipe) error {
	// First, remove the db entry.
	if _, err := db.Exec("DELETE FROM recipes WHERE id == ?", r.Id); err != nil {
		return err
	}

	// Then, remove the recipe XML file.
	if err := os.Remove(path.Join(db.rootdir, r.File)); err != nil {
		return err
	}

	// Finally try removing its directory (if empty).
	os.Remove(path.Dir(path.Join(db.rootdir, r.File)))

	return nil
}
