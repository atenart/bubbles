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
	"os"
	"path"

	"github.com/atenart/bubbles/beerxml"
)

// Retrieve a single ingredient given its id.
func (db *DB) GetIngredient(id int64) (*Ingredient, error) {
	var i Ingredient
	err := db.QueryRow("SELECT * FROM ingredients WHERE id == $1", id).
		Scan(&i.Id, &i.UserId, &i.Name, &i.Type, &i.File)
	if err != nil {
		return nil, err
	}

	if err := db.importIngredientXML(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

// Retrieve the ingredients for a given user.
func (db *DB) GetUserIngredients(uid int64) ([]*Ingredient, error) {
	row, err := db.Query("SELECT * FROM ingredients WHERE user_id == ?", uid)
	if err != nil {
		return nil, err
	}

	var ingredients []*Ingredient
	for row.Next() {
		var i Ingredient
		row.Scan(&i.Id, &i.UserId, &i.Name, &i.Type, &i.File)

		if err := db.importIngredientXML(&i); err != nil {
			return nil, err
		}

		ingredients = append(ingredients, &i)
	}

	return ingredients, nil
}

// Import an ingredient XML file.
func (db *DB) importIngredientXML(i *Ingredient) error {
	var err error
	switch i.Type {
	case "fermentable":
		var f beerxml.Fermentable
		err = db.importXML(i.File, &f)
		i.XML = &f
	case "hop":
		var h beerxml.Hop
		err = db.importXML(i.File, &h)
		i.XML = &h
	case "yeast":
		var y beerxml.Yeast
		err = db.importXML(i.File, &y)
		i.XML = &y
	default:
		return fmt.Errorf("Unknown type.")
	}

	return err
}

// Add a new ingredient.
func (db *DB) AddIngredient(i *Ingredient) error {
	var err error
	if i.File, err = db.newUniqFile(); err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO ingredients (user_id, name, type, file)
VALUES (?, ?, ?, ?)`, i.UserId, i.Name, i.Type, i.File)
	if err != nil {
		return err
	}

	return beerxml.Export(i.XML, path.Join(db.rootdir, i.File))
}

// Update an ingredient.
func (db *DB) UpdateIngredient(i *Ingredient) error {
	_, err := db.Exec(`
REPLACE INTO ingredients (id, user_id, name, type, file)
VALUES (?, ?, ?, ?, ?)`, i.Id, i.UserId, i.Name, i.Type, i.File)
	if err != nil {
		return err
	}

	return beerxml.Export(i.XML, path.Join(db.rootdir, i.File))
}

// Delete an ingredient.
func (db *DB) DeleteIngredient(i *Ingredient) error {
	// First, remove the db entry
	if _, err := db.Exec("DELETE FROM ingredients WHERE id == ?", i.Id); err != nil {
		return err
	}

	// Then, remove the recipe XML file.
	if err := os.Remove(path.Join(db.rootdir, i.File)); err != nil {
		return err
	}

	// Finally try removing its directory (if empty).
	os.Remove(path.Dir(path.Join(db.rootdir, i.File)))

	return nil
}
