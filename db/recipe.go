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
	"io"
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

	db.importXML(&r)

	return &r, nil
}

// Retrieves all recipes for a given user.
func (db *DB) GetUserRecipes(uid int64) ([]*Recipe, error) {
	row, err := db.Query("SELECT * FROM recipes WHERE user_id == ?", uid)
	if err != nil {
		return nil, err
	}

	var recipes []*Recipe
	for row.Next() {
		var r Recipe
		row.Scan(&r.Id, &r.UserId, &r.Name, &r.File, &r.Public)

		recipes = append(recipes, &r)
		db.importXML(&r)
	}

	return recipes, nil
}

// Import a BeerXML recipe associated with a Recipe.
func (db *DB) importXML(recipe *Recipe) error {
	if err := beerxml.Import(path.Join(db.rootdir, recipe.File), &recipe.XML); err == io.EOF {
		recipe.XML = &beerxml.Recipe{}
	} else if err != nil {
		return err
	}

	return nil
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

// Save a new recipe.
func (db *DB) AddRecipe(r *Recipe) (int64, error) {
	var err error
	if r.File, err = db.newUniqFile(); err != nil {
		return -1, err
	}

	result, err := db.Exec(`
INSERT INTO recipes (user_id, name, file, public)
VALUES (?, ?, ?, ?)`, r.UserId, r.Name, r.File, r.Public)
	if err != nil {
		return -1, err
	}

	// Get the generated auto-inc id in response to the previous command.
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	if err := beerxml.Export(&r.XML, path.Join(db.rootdir, r.File)); err != nil {
		return -1, err
	}

	return id, nil
}

// Update a recipe and returns its id.
func (db *DB) UpdateRecipe(r *Recipe) (int64, error) {
	_, err := db.Exec(`
REPLACE INTO recipes (id, user_id, name, file, public)
VALUES (?, ?, ?, ?, ?)`, r.Id, r.UserId, r.Name, r.File, r.Public)
	if err != nil {
		return -1, err
	}

	if err := beerxml.Export(&r.XML, path.Join(db.rootdir, r.File)); err != nil {
		return -1, err
	}

	return r.Id, nil
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
