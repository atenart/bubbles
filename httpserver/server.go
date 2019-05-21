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
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	"github.com/atenart/bubbles/db"
	"github.com/atenart/bubbles/i18n"
)

// Represents a server instance (there is usually a single one).
type Server struct {
	db           *db.DB
	i18n         *i18n.Bundle
	mux          *mux.Router
	templates    *template.Template
	cookie       *securecookie.SecureCookie
	flags        struct {
		// Runtime options
		signUp       bool
		verification bool
		// Debugging options
		debug        bool
		skipLogin    bool
	}
}

// Starts a new server instance.
func Serve(bind string, db *db.DB, i18n *i18n.Bundle, noSignUp, debug, skipLogin bool) error {
	s := &Server{
		db:           db,
		i18n:         i18n,
		mux:          mux.NewRouter(),
		templates:    template.New("templates"),
		// FIXME: Keys are not persistent.
		cookie:       securecookie.New(securecookie.GenerateRandomKey(64),
					       securecookie.GenerateRandomKey(32)),
	}

	s.flags.signUp = !noSignUp
	// Not supported yet.
	s.flags.verification = false
	s.flags.debug = debug
	s.flags.skipLogin = skipLogin

	// i18n: add dummy L function, will be overriden before serving the
	// templates.
	s.templates.Funcs(template.FuncMap{
		"pageName": func() string {
			return ""
		},
		"L": func(id string) string {
			return id
		},
	})

	// Parse the templates.
	templates := []string{
		"httpserver/ui/tpl/*.html",
		"httpserver/ui/tpl/partial/*.html",
	}
	for _, path := range templates {
		if _, err := s.templates.ParseGlob(path); err != nil {
			return err
		}
	}

	// Install unauthenticated mux handlers.
	s.mux.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("httpserver/ui/static"))))
	s.mux.HandleFunc("/sign-up", s.signUp).Methods("POST")
	s.mux.HandleFunc("/activate/{Token:[a-zA-Z0-9]+}", s.activate)
	s.mux.HandleFunc("/login", s.login).Methods("POST")
	s.mux.HandleFunc("/logout", s.logout).Methods("POST")

	// Install authenticated mux handlers.
	s.handleFunc("/", s.index)
	s.handleFunc("/recipe/new", s.newRecipe)
	s.handleFunc("/recipe/{Id:[0-9]+}", s.recipe)
	s.handleFunc("/recipe/{Id:[0-9]+}/{Action:[a-z-]+}", s.saveRecipe).Methods("POST")
	s.handleFunc("/recipe/{Id:[0-9]+}/{Action:[a-z-]+}/{Item:[0-9]+}", s.saveRecipe).Methods("POST")
	s.handleFunc("/account", s.account)
	s.handleFunc("/account/save", s.saveAccount).Methods("POST")
	s.handleFunc("/account/delete", s.deleteAccount).Methods("POST")
	s.handleFunc("/account/export", s.exportData)
	s.handleFunc("/inventory", s.inventory)
	s.handleFunc("/inventory/{Action:[a-z-]+}", s.saveInventory).Methods("POST")
	s.handleFunc("/inventory/{Action:[a-z-]+}/{Item:[0-9]+}", s.saveInventory).Methods("POST")

	// Start serving over HTTP.
	return http.ListenAndServe(bind, s.mux)
}

// Mux HandleFunc wrapper.
func (s *Server) handleFunc(path string, handler func(http.ResponseWriter, *http.Request, *db.User)) *mux.Route {
	return s.mux.HandleFunc(path, s.sessionHandler(handler))
}

// HTTP handler wrapper for user session enforcement.
func (s *Server) sessionHandler(fn func(http.ResponseWriter, *http.Request, *db.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.flags.skipLogin {
			user, err := s.db.GetUserById(1)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			fn(w, r, user)
			return
		}

		// Check if a session cookie is available.
		cookie, err := r.Cookie("session")
		if err != nil {
			s.loginPage(w)
			return
		}

		// Try getting the uid out of the session cookie.
		var uid int64
		if err = s.cookie.Decode("session", cookie.Value, &uid); err != nil {
			s.loginPage(w)
			return
		}

		// Retrive the user info.
		user, err := s.db.GetUserById(uid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Jump to the real handler.
		fn(w, r, user)
	}
}

func (s *Server) loginPage(w http.ResponseWriter) {
	s.executeTemplate(w, nil, "login.html", struct{
		SignUp bool
	}{
		s.flags.signUp,
	})
}

func (s *Server) executeTemplate(w http.ResponseWriter, user *db.User, name string, data interface{}) {
	// Clone the templates. This is done to have a per user i18n translation.
	clone, err := s.templates.Clone()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	if user != nil {
		// Get a new localizer matching the user's lang.
		l := s.i18n.Localizer(user.Lang)

		// Set the i18n func.
		clone.Funcs(template.FuncMap{
			"pageName": func() string {
				return name
			},
			"L": func(id string) string {
				return l.Localize(id)
			},
		})
	}

	clone.ExecuteTemplate(w, name, data)
}
