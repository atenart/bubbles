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
	"encoding/base64"
	"golang.org/x/crypto/scrypt"

	_ "github.com/mattn/go-sqlite3"
)

// Hashes a plaintext password.
func (db *DB) HashPassword(password string) (string, error) {
	// TODO: check & document.
	dk, err := scrypt.Key([]byte(password), db.salt, 1<<15, 8, 1, 32)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(dk), nil
}

// Retrieves an user information from the db, given an email.
func (db *DB) GetUserByEmail(email string) (*User, error) {
	var u User
	err := db.QueryRow("SELECT * FROM users WHERE email == $1", email).
		Scan(&u.Id, &u.Email, &u.Password, &u.RegistrationDate, &u.Token,
		     &u.Enabled, &u.Lang)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Retrieves an user information from the db, given an uid.
func (db *DB) GetUserById(uid int64) (*User, error) {
	var u User
	err := db.QueryRow("SELECT * FROM users WHERE id == $1", uid).
		Scan(&u.Id, &u.Email, &u.Password, &u.RegistrationDate, &u.Token,
		     &u.Enabled, &u.Lang)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Adds a new user.
func (db *DB) AddUser(name, password, token string, verification bool) error {
	_, err := db.Exec(`
INSERT INTO users (email, password, token, enabled)
VALUES (?, ?, ?, ?)
`, name, password, token, !verification)
	return err
}

// Activate an user based on a provided token.
func (db *DB) ActivateUser(token string) error {
	_, err := db.Exec("UPDATE users SET enabled = 1 WHERE token == ?", token)
	return err
}

// Update an user info.
func (db *DB) UpdateUser(u *User) error {
	_, err := db.Exec(`
REPLACE INTO users (id, email, password, token, enabled, lang)
VALUES (?, ?, ?, ?, ?, ?)`, u.Id, u.Email, u.Password, u.Token, u.Enabled, u.Lang)
	return err
}

// Delete an user and all its data.
func (db *DB) DeleteUser(u *User) error {
	// Delete all the ingredients associated to the user.
	ingredients, err := db.GetUserIngredients(u.Id)
	if err != nil {
		return err
	}

	for _, i := range ingredients {
		if err := db.DeleteIngredient(i); err != nil {
			return err
		}
	}

	// Delete all the recipes associated to the user.
	recipes, err := db.GetUserRecipes(u.Id)
	if err != nil {
		return err
	}

	for _, r := range recipes {
		if err := db.DeleteRecipe(r); err != nil {
			return err
		}
	}

	// Now, delete the user itself. Do this at the end: if something went
	// bad, the user can still sign in to report the issue.
	if _, err := db.Exec("DELETE FROM users WHERE id == ?", u.Id); err != nil {
		return err
	}

	return nil
}
