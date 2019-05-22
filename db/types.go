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

import "github.com/atenart/bubbles/beerxml"

// Represents an user.
type User struct {
	Id               int64
	Email            string
	Password         string
	RegistrationDate string
	Token            string
	Enabled          bool
	Lang             string
}

// Represents a recipe and contains a path to its associated BeerXML file.
type Recipe struct {
	Id      int64
	UserId  int64
	Name    string
	File    string
	Public  bool
	XML     *beerxml.Recipe
}

// Represents an ingredient (fermentable, hops, yeats, ...) in an user inventory
// and contains a path to its associated BeerXML file.
type Ingredient struct {
	Id     int64
	UserId int64
	Name   string
	Type   string
	Link   string
	File   string
	XML    interface{}
}
