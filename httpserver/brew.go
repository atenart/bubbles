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
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/atenart/bubbles/beerxml"
	"github.com/atenart/bubbles/db"
)

// Brews listing.
func (s *Server) brews(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Retrieve all the user's brews.
	brews, err := s.db.GetUserBrews(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Retrieve step names (FIXME).
	var names []string
	for _, b := range brews {
		n, _ := db.StepName(b.Step)
		names = append(names, n)
	}

	s.executeTemplate(w, user, "brews.html", struct{
		CSRF      template.HTML
		Title     string
		Brews     []*db.Brew
		StepNames []string
	}{
		csrf.TemplateField(r),
		"Bubbles - brews",
		brews,
		names,
	})
}

// Display a brew.
func (s *Server) brew(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	brew, err := s.db.GetBrew(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if brew.UserId != user.Id {
		http.Error(w, "Permission denied to edit this brew.", 500)
		return
	}

	ingredients := &beerxml.BeerXML{}
	if brew.Step == db.StepPrepare {
		ingredients = addUpIngredients(brew.XML)
	}

	dryHop := false
	for _, h := range brew.XML.Hops {
		if h.Use == "Dry hop" {
			dryHop = true
			break
		}
	}

	name, desc := db.StepName(brew.Step)
	s.executeTemplate(w, user, "brew.html", struct{
		CSRF        template.HTML
		Title       string
		StepName    string
		StepDesc    string
		MaxStep     int
		Brew        *db.Brew
		Calc        *Calculation
		Ingredients *beerxml.BeerXML
		DryHop      bool
	}{
		csrf.TemplateField(r),
		fmt.Sprintf("Bubbles - brew/%s %s", brew.XML.Name, brew.XML.Date),
		name,
		desc,
		db.StepMax,
		brew,
		calculations(brew.XML),
		ingredients,
		dryHop,
	})
}

func addUpIngredients(recipe *beerxml.Recipe) *beerxml.BeerXML {
	var ingredients beerxml.BeerXML

	// Add up fermentables.
	seen := make(map[string]int)
	for _, f := range recipe.Fermentables {
		if _, ok := seen[f.Name]; !ok {
			ingredients.Fermentables = append(ingredients.Fermentables, f)
			seen[f.Name] = len(ingredients.Fermentables) - 1
			continue
		}
		ingredients.Fermentables[seen[f.Name]].Amount += f.Amount
	}
	sort(ingredients.Fermentables)

	// Add up hops.
	seen = make(map[string]int)
	for _, h := range recipe.Hops {
		if _, ok := seen[h.Name]; !ok {
			ingredients.Hops = append(ingredients.Hops, h)
			seen[h.Name] = len(ingredients.Hops) - 1
			continue
		}
		ingredients.Hops[seen[h.Name]].Amount += h.Amount
	}
	sort(ingredients.Hops)

	// Add up yeasts.
	seen = make(map[string]int)
	for _, y := range recipe.Yeasts {
		if _, ok := seen[y.Name]; !ok {
			ingredients.Yeasts = append(ingredients.Yeasts, y)
			seen[y.Name] = len(ingredients.Yeasts) - 1
			continue
		}
		ingredients.Yeasts[seen[y.Name]].Amount += y.Amount
	}
	sort(ingredients.Yeasts)

	return &ingredients
}

func (s *Server) getBrew(id, uid int64) (*db.Brew, error) {
	brew, err := s.db.GetBrew(id)
	if err != nil {
		return nil, err
	}

	// Check user rights.
	if brew.UserId != uid {
		return nil, fmt.Errorf("Permission denied to edit this brew")
	}

	return brew, nil
}

// Delete a brew.
func (s *Server) deleteBrew(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	brew, err := s.getBrew(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.db.DeleteBrew(brew); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/brews", 302)
}

// Save information of a brew.
func (s *Server) brewSave(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	brew, err := s.getBrew(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form field.", 500)
		return
	}

	switch mux.Vars(r)["Action"] {
	case "fermentation":
		brew.XML.OG, _ = strconv.ParseFloat(r.FormValue("og"), 64)
	case "bottling":
		brew.XML.FG, _ = strconv.ParseFloat(r.FormValue("fg"), 64)

		// Update calc params.
		brew.XML.ABV = math.Round(brew.XML.CalcRealABV() * 10) / 10
	case "done":
		brew.XML.Notes = r.FormValue("notes")
		brew.XML.TasteNotes = r.FormValue("taste-notes")
	default:
		http.Error(w, "Unknown action.", 500)
		return
	}

	if err := s.db.UpdateBrew(brew); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/brew/%d", id), 302)
}

// Move brew to the previous step.
func (s *Server) brewPrevStep(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Retrieve the brew info
	brew, err := s.getBrew(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Increment the brew step.
	if brew.Step > 0 {
		brew.Step--
	}

	if err := s.db.UpdateBrew(brew); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/brew/%d", id), 302)
}

// Move brew to the next step.
func (s *Server) brewNextStep(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Retrieve the brew info
	brew, err := s.getBrew(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// If we start the brew (moving from step 0 to 1), use the current date
	// as the brew date.
	if brew.Step == 0 && brew.XML.Date == "" {
		// TODO: local time?
		brew.XML.Date = time.Now().UTC().Format("02 Jan 2006")
	}

	// Increment the brew step.
	if brew.Step < db.StepMax {
		brew.Step++
	}

	if err := s.db.UpdateBrew(brew); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/brew/%d", id), 302)
}

// Create a new brew from a recipe.
func (s *Server) newBrew(w http.ResponseWriter, r *http.Request, user *db.User) {
	id, err := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Retrieve the recipe current info.
	recipe, err := s.getRecipe(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	brew := &db.Brew{
		UserId:   user.Id,
		RecipeId: recipe.Id,
		Step:     db.StepPrepare,
		XML:      recipe.XML,
	}

	newId, err := s.db.AddBrew(brew)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/brew/%d", newId), 302)
}
