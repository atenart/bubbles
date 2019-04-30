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

	"github.com/gorilla/mux"
	"github.com/atenart/bubbles/db"
)

// Sets a secure cookie (unless the server runs in debug mode).
func (s *Server) setSecureCookie(w http.ResponseWriter, name, value string, maxAge int) {
	secure := true

	// When running in debug mode, do not set the secure flag as the server
	// might run locally and be accessed over HTTP.
	if s.flags.debug {
		secure = false
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Login an user: makes checks, validate user/password & set a secure cookie.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	// Get the user & password from the POSTed form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form field.", 500)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Try retrieving an (enabled) existing user.
	user, err := s.db.GetUserByEmail(email)
	if err != nil || !user.Enabled {
		// TODO: give a meaningfull feedback to the user.
		http.Redirect(w, r, "/", 302)
		return
	}

	// Generate an hash out of the provided plaintext password.
	hash, err := s.db.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Check the provided password matches the one we have.
	if strings.Compare(user.Password, hash) != 0 {
		// TODO: give a meaningfull feedback to the user.
		http.Redirect(w, r, "/", 302)
		return
	}

	// Create an encoded data for the session cookie and set it.
	payload, err := s.cookie.Encode("session", user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.setSecureCookie(w, "session", payload, 0)

	http.Redirect(w, r, "/", 302)
}

// Logout an user by setting an empty session cookie.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	// Delete the session cookie by settings its max age to -1.
	s.setSecureCookie(w, "session", "", -1)

	http.Redirect(w, r, "/", 302)
}

// Sign-up a new user.
func (s *Server) signUp(w http.ResponseWriter, r *http.Request) {
	if !s.flags.signUp {
		http.Redirect(w, r, "/", 302)
	}

	// Get the user & password from the POSTed form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form field.", 500)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/", 302)
		return
	}

	// Compute the hash password out of the provided plaintext one.
	hash, err := s.db.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Generate a random token for the user activation.
	token := db.GenToken(32)

	// Try adding the new user.
	if err := s.db.AddUser(email, hash, token, s.flags.verification); err != nil {
		// TODO: could be the user already exists.
		http.Error(w, err.Error(), 500)
		return
	}

	// TODO: send an email with the activation link & give user feedack.
	http.Redirect(w, r, "/", 302)
}

// Activates an user if the provided token in the url matches.
func (s *Server) activate(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["Token"]

	if err := s.db.ActivateUser(token); err != nil {
		// TODO: provide more feedback.
	}

	// TODO: provide more feedback.
	http.Redirect(w, r, "/", 302)
}
