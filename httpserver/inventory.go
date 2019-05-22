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

package httpserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/atenart/bubbles/db"
	"github.com/atenart/bubbles/beerxml"
	"github.com/gorilla/mux"
)

// Inventory page (per-user).
func (s *Server) inventory(w http.ResponseWriter, r *http.Request, user *db.User) {
	ingredients, err := s.db.GetUserIngredients(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	sort(ingredients)

	s.executeTemplate(w, user, "inventory.html", struct{
		Title	    string
		Ingredients []*db.Ingredient
	}{
		"Bubbles - inventory",
		ingredients,
	})
}

// Save an inventory item.
func (s *Server) saveInventory(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Retrieve elements from the POSTed form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form field.", 500)
		return
	}

	action := mux.Vars(r)["Action"]
	if v, err := strconv.ParseInt(mux.Vars(r)["Item"], 10, 32); err == nil {
		// Retrieve the ingredient to modify.
		ingredient, err := s.db.GetIngredient(int64(v))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Check the ingredient belongs to the current user.
		if ingredient.UserId != user.Id {
			http.Error(w, "Access to ingredient denied", 500)
			return
		}

		// Perform the requested action.
		switch action {
		case "del":
			if err := s.db.DeleteIngredient(ingredient); err != nil {
				http.Error(w, err.Error(), 500)
			}
		case "edit-fermentable":
			fallthrough
		case "edit-hop":
			fallthrough
		case "edit-yeast":
			if err := s.editIngredient(r, user, ingredient); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		default:
			http.Error(w, "Unknown action", 500)
			return
		}
	} else {
		if err := s.addIngredient(r, user, action); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	http.Redirect(w, r, "/inventory", 302)
}

// Add a new ingredient to an user inventory.
func (s *Server) addIngredient(r *http.Request, user *db.User, action string) error {
	i := &db.Ingredient{
		UserId: user.Id,
	}

	switch action {
	case "add-fermentable":
		var f beerxml.Fermentable
		if err := formToFermentable(r, &f); err != nil {
			return err
		}

		i.Name = f.Name
		i.Type = "fermentable"
		i.Link = r.FormValue("link")
		i.XML = &f
	case "add-hop":
		var h beerxml.Hop
		if err := formToHop(r, &h); err != nil {
			return err
		}

		i.Name = h.Name
		i.Type = "hop"
		i.Link = r.FormValue("link")
		i.XML = &h
	case "add-yeast":
		var y beerxml.Yeast
		if err := formToYeast(r, &y); err != nil {
			return err
		}

		i.Name = y.Name
		i.Type = "yeast"
		i.Link = r.FormValue("link")
		i.XML = &y
	default:
		return fmt.Errorf("Unknown ingredient")
	}

	return s.db.AddIngredient(i)
}

// Edit an ingredient.
func (s *Server) editIngredient(r *http.Request, user *db.User, i *db.Ingredient) error {
	var err error
	switch elmt := i.XML.(type) {
	case *beerxml.Fermentable:
		err = formToFermentable(r, elmt)
	case *beerxml.Hop:
		err = formToHop(r, elmt)
	case *beerxml.Yeast:
		err = formToYeast(r, elmt)
	default:
		err = fmt.Errorf("Unkonwn type.")
	}
	if err != nil {
		return err
	}

	i.Link = r.FormValue("link")

	return s.db.UpdateIngredient(i)
}
