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

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/atenart/bubbles/db"
	"github.com/atenart/bubbles/i18n"
	"github.com/atenart/bubbles/sendmail"
)

// Represents a server instance (there is usually a single one).
type Server struct {
	URL          string
	db           *db.DB
	sendmail     *sendmail.Sendmail
	i18n         *i18n.Bundle
	mux          *mux.Router
	templates    *template.Template
	cookie       *securecookie.SecureCookie
	uploadMax    int64
	flags        struct {
		// Runtime options
		signUp       bool
		verification bool
		// Debugging options
		debug        bool
	}
}

// Starts a new server instance.
func Serve(bind, url string, db *db.DB, sendmail *sendmail.Sendmail, i18n *i18n.Bundle,
	   noSignUp, noVerification, debug  bool) error {
	s := &Server{
		URL:          url,
		db:           db,
		sendmail:     sendmail,
		i18n:         i18n,
		mux:          mux.NewRouter(),
		templates:    template.New("templates"),
		uploadMax:    10 << 20, // 10 MB
	}

	s.flags.signUp = !noSignUp
	s.flags.verification = !noVerification
	s.flags.debug = debug

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
	s.mux.HandleFunc("/robots.txt", s.robotstxt)
	s.mux.HandleFunc("/sign-up", s.signUp).Methods("POST")
	s.mux.HandleFunc("/activate/{Token:[a-zA-Z0-9]+}", s.activate)
	s.mux.HandleFunc("/login", s.login).Methods("POST")
	s.mux.HandleFunc("/logout", s.logout)

	// Install authenticated mux handlers.
	s.handleFunc("/", s.index)
	s.handleFunc("/recipes", s.recipes)
	s.handleFunc("/recipe/new", s.newRecipe)
	s.handleFunc("/recipe/clone/{Id:[0-9]+}", s.cloneRecipe)
	s.handleFunc("/recipe/{Id:[0-9]+}", s.recipe)
	s.handleFunc("/recipe/{Id:[0-9]+}/{Action:[a-z-]+}", s.saveRecipe).Methods("POST")
	s.handleFunc("/recipe/{Id:[0-9]+}/{Action:[a-z-]+}/{Item:[0-9]+}", s.saveRecipe).Methods("POST")
	s.handleFunc("/account", s.account)
	s.handleFunc("/account/save", s.saveAccount).Methods("POST")
	s.handleFunc("/account/delete", s.deleteAccount).Methods("POST")
	s.handleFunc("/account/export", s.exportData)
	s.handleFunc("/account/import", s.importData).Methods("POST")
	s.handleFunc("/inventory", s.inventory)
	s.handleFunc("/inventory/{Action:[a-z-]+}", s.saveInventory).Methods("POST")
	s.handleFunc("/inventory/{Action:[a-z-]+}/{Item:[0-9]+}", s.saveInventory).Methods("POST")
	s.handleFunc("/brews", s.brews)
	s.handleFunc("/brew/new/{Id:[0-9]+}", s.newBrew)
	s.handleFunc("/brew/{Id:[0-9]+}", s.brew)
	s.handleFunc("/brew/{Id:[0-9]+}/delete", s.deleteBrew).Methods("POST")
	s.handleFunc("/brew/{Id:[0-9]+}/prev", s.brewPrevStep).Methods("POST")
	s.handleFunc("/brew/{Id:[0-9]+}/next", s.brewNextStep).Methods("POST")
	s.handleFunc("/brew/{Id:[0-9]+}/save-{Action:[a-z-]+}", s.brewSave).Methods("POST")

	// Setup secure cookie.
	s.cookie = securecookie.New(s.db.LoadKey("hash.securecookie", 64),
				    s.db.LoadKey("block.securecookie", 32))

	// Instantiate the CSRF protection.
	rf := csrf.Protect(s.db.LoadKey("csrf", 32), csrf.Secure(!s.flags.debug))

	// Start serving over HTTP.
	return http.ListenAndServe(bind, rf(s.mux))
}

// Index.
func (s *Server) index(w http.ResponseWriter, r *http.Request, user *db.User) {
	http.Redirect(w, r, "/recipes", 302)
}

// Mux HandleFunc wrapper.
func (s *Server) handleFunc(path string, handler func(http.ResponseWriter, *http.Request, *db.User)) *mux.Route {
	return s.mux.HandleFunc(path, s.sessionHandler(handler))
}

// HTTP handler wrapper for user session enforcement.
func (s *Server) sessionHandler(fn func(http.ResponseWriter, *http.Request, *db.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if a session cookie is available.
		cookie, err := r.Cookie("session")
		if err != nil {
			s.loginPage(w, r)
			return
		}

		// Try getting the uid out of the session cookie.
		var uid int64
		if err = s.cookie.Decode("session", cookie.Value, &uid); err != nil {
			s.loginPage(w, r)
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

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", 302)
		return
	}

	s.executeTemplate(w, nil, "login.html", struct{
		CSRF         template.HTML
		SignUp       bool
		Verification bool
	}{
		csrf.TemplateField(r),
		s.flags.signUp,
		s.flags.verification,
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

func (s *Server) robotstxt(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `User-agent: *
Allow: /$
Disallow: /
`)
}
