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
	"strings"

	"github.com/atenart/bubbles/db"
)

// Account page (per-user).
func (s *Server) account(w http.ResponseWriter, r *http.Request, user *db.User) {
	s.executeTemplate(w, user, "account.html", struct{
		Title	string
		User    *db.User
		Tags    []string
	}{
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
