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
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/atenart/bubbles/db"
	"github.com/atenart/bubbles/beerxml"
)

// Account page (per-user).
func (s *Server) account(w http.ResponseWriter, r *http.Request, user *db.User) {
	s.executeTemplate(w, user, "account.html", struct{
		CSRF	template.HTML
		Title	string
		User    *db.User
		Tags    []string
	}{
		csrf.TemplateField(r),
		"Bubbles - account",
		user,
		s.i18n.Tags(),
	})
}

// Update an account's info.
func (s *Server) saveAccount(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Retrieve elements from the POSTed form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	currentPassword := r.FormValue("current-password")
	newPassword := r.FormValue("new-password")
	confirmPassword := r.FormValue("confirm-password")

	user.Lang = r.FormValue("lang")

	// Password udate
	if currentPassword != "" && newPassword != "" && confirmPassword != "" {
		currentHash, err := s.db.HashPassword(currentPassword)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if strings.Compare(user.Password, currentHash) != 0 {
			http.Error(w, "Wrong password", 500)
			return
		}

		if strings.Compare(newPassword, confirmPassword) != 0 {
			http.Error(w, "New passwords do not match", 500)
			return
		}

		user.Password, err = s.db.HashPassword(newPassword)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err := s.db.UpdateUser(user); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/account", 302)
}

// Delete an user account and all its data.
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request, user *db.User) {
	if err := s.db.DeleteUser(user); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.logout(w, r)
}

// Export an user data (recipes & inventory) into a single BeerXML file.
func (s *Server) exportData(w http.ResponseWriter, r *http.Request, user *db.User) {
	recipes, err := s.db.GetUserRecipes(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ingredients, err := s.db.GetUserIngredients(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var xml beerxml.BeerXML
	for _, r := range recipes {
		beerxml.InsertToXML(&xml, r.XML)
	}
	for _, i := range ingredients {
		beerxml.InsertToXML(&xml, i.XML)
	}

	w.Header().Add("Content-Type", "text/xml")
	w.Header().Set("Content-Disposition",
		       fmt.Sprintf("attachment; filename=bubbles_%s.xml",
				   time.Now().UTC().Format("200601021504")))

	if err := beerxml.Export(&xml, w); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// Import an user data (recipes & inventory) into the DB.
func (s *Server) importData(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Parse the POSTed form.
	if err := r.ParseMultipartForm(s.uploadMax); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Retrieve the file to load.
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	// Try parsing it to a BeerXML object.
	var xml beerxml.BeerXML
	if err := beerxml.Import(file, &xml); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Add all the recipes.
	for _, r := range xml.Recipes {
		recipe := &db.Recipe{
			Name:   r.Name,
			UserId: user.Id,
			XML:    &r,
		}

		if _, err := s.db.AddRecipe(recipe); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	// Add all the ingredients.
	// We try our best, but do not report errors if the ingredient already
	// exists in the db.
	for _, f := range xml.Fermentables {
		ingredient := &db.Ingredient{
			Name:   f.Name,
			UserId: user.Id,
			Type:   "fermentable",
			XML:    &f,
		}

		s.db.AddIngredient(ingredient)
	}
	for _, h := range xml.Hops {
		ingredient := &db.Ingredient{
			Name:   h.Name,
			UserId: user.Id,
			Type:   "hop",
			XML:    &h,
		}

		s.db.AddIngredient(ingredient)
	}
	for _, y := range xml.Yeasts {
		ingredient := &db.Ingredient{
			Name:   y.Name,
			UserId: user.Id,
			Type:   "yeast",
			XML:    &y,
		}

		s.db.AddIngredient(ingredient)
	}

	http.Redirect(w, r, "/account", 302)
}
