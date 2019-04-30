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

	"github.com/atenart/bubbles/beerxml"
	"github.com/atenart/bubbles/db"
	"github.com/gorilla/mux"
)

// Return a *db.Recipe object, from the database if it exists or new if it
// doesn't.
func (s *Server) getRecipe(id, uid int64) (*db.Recipe, error) {
	// Retrive an existing recipe.
	recipe, err := s.db.GetRecipe(id)
	if err != nil {
		return nil, err
	}

	// Check the recipe belongs to the current user.
	if recipe.UserId != uid {
		// TODO: give more feedback.
		return nil, fmt.Errorf("Permission denied to edit this recipe.")
	}

	return recipe, nil
}

// Display a recipe edition page.
func (s *Server) recipe(w http.ResponseWriter, r *http.Request, user *db.User) {
	var err error

	id, _ := strconv.ParseInt(mux.Vars(r)["Id"], 10, 64)

	// Retrieve the recipe current info.
	recipe, err := s.getRecipe(id, user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.executeTemplate(w, user, "recipe.html", struct{
		Title  string
		User   *db.User
		Recipe *db.Recipe
		Styles *[]beerxml.Style
		Calc   *Calculation
	}{
		fmt.Sprintf("Bubbles - recipe/%s", recipe.Name),
		user,
		recipe,
		s.db.Styles,
		calculations(recipe),
	})
}

type Cursor struct {
	Name     string
	Val      float64
	Min, Max float64
	Cursor   float64
	ValOK    bool
	RGB      template.CSS
}

type Calculation struct {
	BoilSize float64
	Cursors  []Cursor
}

// Compute the cursor margin given a value and to boundaries.
func cursor(val, min, max float64) float64 {
	if val <= min {
		return 0
	}
	if val >= max {
		return 100
	}
	return ((val - min) * 100) / (max - min)
}

// Retrieve all the calculation for a given recipe.
func calculations(r *db.Recipe) *Calculation {
	og := r.XML.CalcOG()
	fg := r.XML.CalcFG()
	abv := r.XML.CalcABV()
	ibu := r.XML.CalcIBU()
	color := r.XML.CalcColor()
	ibuOg := r.XML.CalcIbuOg()
	ibuRe := r.XML.CalcIbuRe()

	key := math.Round(color * 10) / 10
	if key <= 0 {
		key = 0.1
	} else if key > 40 {
		key = 40
	}
	hex := beerxml.SrmHex[key]

	return &Calculation{
		BoilSize: math.Round(r.XML.CalcBoilSize() * 10) / 10,
		Cursors: []Cursor{
			{
				"OG",
				math.Round(og * 1000) / 1000,
				r.XML.Style.OgMin,
				r.XML.Style.OgMax,
				cursor(og, r.XML.Style.OgMin, r.XML.Style.OgMax),
				(r.XML.Style.OgMin <= og && og <= r.XML.Style.OgMax),
				"",
			},
			{
				"FG",
				math.Round(fg * 1000) / 1000,
				r.XML.Style.FgMin,
				r.XML.Style.FgMax,
				cursor(fg, r.XML.Style.FgMin, r.XML.Style.FgMax),
				(r.XML.Style.FgMin <= fg && fg <= r.XML.Style.FgMax),
				"",
			},
			{
				"ABV",
				math.Round(abv * 10) / 10,
				r.XML.Style.AbvMin,
				r.XML.Style.AbvMax,
				cursor(abv, r.XML.Style.AbvMin, r.XML.Style.AbvMax),
				(r.XML.Style.AbvMin <= abv && abv <= r.XML.Style.AbvMax),
				"",
			},
			{
				"IBU",
				math.Round(ibu * 100) / 100,
				r.XML.Style.IbuMin,
				r.XML.Style.IbuMax,
				cursor(ibu, r.XML.Style.IbuMin, r.XML.Style.IbuMax),
				(r.XML.Style.IbuMin <= ibu && ibu <= r.XML.Style.IbuMax),
				"",
			},
			{
				"Color",
				math.Round(color * 100) / 100,
				r.XML.Style.ColorMin,
				r.XML.Style.ColorMax,
				cursor(color, r.XML.Style.ColorMin, r.XML.Style.ColorMax),
				(r.XML.Style.ColorMin <= color && color <= r.XML.Style.ColorMax),
				template.CSS(fmt.Sprintf("rgb(%d, %d, %d)", hex.R, hex.G, hex.B)),
			},
			{
				"IBU/OG",
				math.Round(ibuOg * 100) / 100,
				0.2,
				1.2,
				cursor(ibuOg, 0.2, 1.2),
				true,
				"",
			},
			{
				"IBU/RE",
				math.Round(ibuRe * 100) / 100,
				0,
				15,
				cursor(ibuOg, 0, 15),
				true,
				"",
			},
		},
	}
}

// Add a new recipe,
func (s *Server) newRecipe(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Recipe does not exist yet.
	recipe := &db.Recipe{
		Name:   "New recipe",
		UserId: user.Id,
		XML:    &beerxml.Recipe{
			Version:    1,
			Efficiency: 70,
		},
	}

	// Save or update recipe.
	id, err := s.db.AddRecipe(recipe)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipe/%d", id), 302)
}

// Save a recipe.
func (s *Server) saveRecipe(w http.ResponseWriter, r *http.Request, user *db.User) {
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

	// Retrieve elements from the POSTed form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form field.", 500)
		return
	}

	action := mux.Vars(r)["Action"]
	var item int
	if v, err := strconv.ParseInt(mux.Vars(r)["Item"], 10, 32); err == nil {
		item = int(v)
	} else {
		item = -1
	}

	switch action {
	// Update a recipe.
	case "save":
		if err := formToRecipe(r, recipe); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		updateStyle(r, recipe, s.db.Styles)
	// Delete a recipe.
	case "delete":
		if err := s.db.DeleteRecipe(recipe); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Override the final redirect, as the recipe do not exist
		// anymore.
		http.Redirect(w, r, "/", 302)
		return
	case "add-fermentable":
		fermentable, err := formToFermentable(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "add-hop":
		hop, err := formToHop(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "add-yeast":
		yeast, err := formToYeast(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, yeast); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "add-mash-step":
		step, err := formToMashStep(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, step); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "save-fermentable":
		fermentable, err := formToFermentable(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Fermentable{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "save-hop":
		hop, err := formToHop(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Hop{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "save-yeast":
		yeast, err := formToYeast(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Yeast{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, yeast); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "save-mash-step":
		mashStep, err := formToMashStep(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.MashStep{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, mashStep); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "del-fermentable":
		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Fermentable{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "del-hop":
		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Hop{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "del-yeast":
		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Yeast{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "del-mash-step":
		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.MashStep{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	default:
		http.Error(w, fmt.Sprintf("Unknown action '%s'", action), 500)
	}

	// Update recipe.
	id, err = s.db.UpdateRecipe(recipe)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipe/%d", id), 302)
}

// Convert elements POSTed from a form into a *db.Recipe.
func formToRecipe(r *http.Request, recipe *db.Recipe) error {
	var err error

	// First convert elements going directly into the db.Recipe.
	recipe.Name = r.FormValue("name")
	if recipe.Public, err = strconv.ParseBool(r.FormValue("public")); err != nil {
		return err
	}

	// Then take care of the BeerXML part.

	recipe.XML.Type = r.FormValue("type")
	recipe.XML.Notes = r.FormValue("notes")

	if version, err := strconv.ParseInt(r.FormValue("version"), 10, 32); err == nil {
		recipe.XML.Version = int32(version)
	}

	recipe.XML.BatchSize, _ = strconv.ParseFloat(r.FormValue("batch-size"), 64)
	recipe.XML.BoilTime, _ = strconv.ParseFloat(r.FormValue("boil-time"), 64)
	recipe.XML.Efficiency, _ = strconv.ParseFloat(r.FormValue("efficiency"), 64)

	// Sanity checks.
	if recipe.Name == "" {
		return fmt.Errorf("'Name' is required.")
	}

	return nil
}

// Update a recipe's style given the style POST'ed.
func updateStyle(r *http.Request, recipe *db.Recipe, styles *[]beerxml.Style) {
	name := r.FormValue("style")

	for _, style := range *styles {
		if name == style.Name {
			recipe.XML.Style = style
		}
	}
}

// Convert elements POSTed from a form into a beerxml.Fermentable.
func formToFermentable(r *http.Request) (*beerxml.Fermentable, error) {
	var fermentable beerxml.Fermentable

	fermentable.Name = r.FormValue("name")
	fermentable.Type = r.FormValue("type")
	fermentable.Yield, _ = strconv.ParseFloat(r.FormValue("yield"), 64)
	fermentable.Color, _ = strconv.ParseFloat(r.FormValue("color"), 64)
	fermentable.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)

	// Sanity checks
	if fermentable.Name == "" {
		return nil, fmt.Errorf("'Name' is required.")
	}

	return &fermentable, nil
}

// Convert elements POSTed from a form into a beerxml.Hop.
func formToHop(r *http.Request) (*beerxml.Hop, error) {
	var hop beerxml.Hop

	hop.Name = r.FormValue("name")
	hop.Form = r.FormValue("form")
	hop.Use = r.FormValue("use")
	hop.Alpha, _ = strconv.ParseFloat(r.FormValue("alpha"), 64)
	hop.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)
	hop.Time, _ = strconv.ParseFloat(r.FormValue("time"), 64)

	// Sanity checks
	if hop.Name == "" {
		return nil, fmt.Errorf("'Name' is required.")
	}

	return &hop, nil
}

// Convert elements POSTed from a form into a beerxml.Yeast.
func formToYeast(r *http.Request) (*beerxml.Yeast, error) {
	var yeast beerxml.Yeast

	yeast.Name = r.FormValue("name")
	yeast.Form = r.FormValue("form")
	yeast.Attenuation, _ = strconv.ParseFloat(r.FormValue("attenuation"), 64)
	yeast.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)

	if r.FormValue("unit") == "kilogram" {
		yeast.AmountIsWeight = true
	}

	// Sanity checks
	if yeast.Name == "" {
		return nil, fmt.Errorf("'Name' is required.")
	}

	return &yeast, nil
}

// Convert elements POSTed from a form into a beerxml.MashStep.
func formToMashStep(r *http.Request) (*beerxml.MashStep, error) {
	var step beerxml.MashStep

	step.Name = r.FormValue("name")
	step.Type = r.FormValue("type")
	step.StepTemp, _ = strconv.ParseFloat(r.FormValue("temperature"), 64)
	step.StepTime, _ = strconv.ParseFloat(r.FormValue("time"), 64)

	// Sanity checks
	if step.Name == "" {
		return nil, fmt.Errorf("'Name' is required.")
	}

	return &step, nil
}
