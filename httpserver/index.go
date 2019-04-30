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
	"net/http"

	"github.com/atenart/bubbles/db"
)

// Main page (per-user).
func (s *Server) index(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Retrieve all the recipes authored by the current user.
	recipes, err := s.db.GetUserRecipes(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.executeTemplate(w, user, "index.html", struct{
		User    *db.User
		Recipes []*db.Recipe
	}{
		user,
		recipes,
	})
}
