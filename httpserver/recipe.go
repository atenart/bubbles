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

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/atenart/bubbles/beerxml"
	"github.com/atenart/bubbles/db"
)

// Recipes listing.
func (s *Server) recipes(w http.ResponseWriter, r *http.Request, user *db.User) {
	// Retrieve all the recipes authored by the current user.
	recipes, err := s.db.GetUserRecipes(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.executeTemplate(w, user, "recipes.html", struct{
		Title   string
		Recipes []*db.Recipe
	}{
		"Bubbles - recipes",
		recipes,
	})
}

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

	// Retrieve the user's ingredients from its inventory.
	ingredients, err := s.db.GetUserIngredients(user.Id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Sort ingredients.
	var fermentables []*beerxml.Fermentable
	var hops []*beerxml.Hop
	var yeasts []*beerxml.Yeast
	for _, i := range ingredients {
		switch xml := i.XML.(type) {
		case *beerxml.Fermentable:
			fermentables = append(fermentables, xml)
		case *beerxml.Hop:
			hops = append(hops, xml)
		case *beerxml.Yeast:
			yeasts = append(yeasts, xml)
		}
	}

	s.executeTemplate(w, user, "recipe.html", struct{
		CSRF         template.HTML
		Title        string
		Recipe       *db.Recipe
		Styles       *[]beerxml.Style
		Calc         *Calculation
		CalcIdx      []string
		Fermentables []*beerxml.Fermentable
		Hops         []*beerxml.Hop
		Yeasts       []*beerxml.Yeast
	}{
		csrf.TemplateField(r),
		fmt.Sprintf("Bubbles - recipe/%s", recipe.Name),
		recipe,
		s.db.Styles,
		calculations(recipe.XML),
		[]string{"OG", "FG", "ABV", "IBU", "Color", "IBU/OG", "IBU/RE"},
		fermentables,
		hops,
		yeasts,
	})
}

type Cursor struct {
	Val      float64
	Min, Max float64
	Cursor   float64
	ValOK    bool
	RGB      template.CSS
}

type Calculation struct {
	VolumeTot float64
	BoilSize  float64
	Cursors   map[string]Cursor
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
func calculations(r *beerxml.Recipe) *Calculation {
	ibuOg := r.CalcIbuOg()
	ibuRe := r.CalcIbuRe()

	key := math.Round(r.EstColor * 10) / 10
	if key <= 0 {
		key = 0.1
	} else if key > 40 {
		key = 40
	}
	hex := beerxml.SrmHex[key]

	return &Calculation{
		VolumeTot: math.Round(r.CalcVolumeTot() * 10) / 10,
		BoilSize: math.Round(r.CalcBoilSize() * 10) / 10,
		Cursors: map[string]Cursor{
			"OG": {
				r.EstOG,
				r.Style.OgMin,
				r.Style.OgMax,
				cursor(r.EstOG, r.Style.OgMin, r.Style.OgMax),
				(r.Style.OgMin <= r.EstOG && r.EstOG <= r.Style.OgMax),
				"",
			},
			"FG": {
				r.EstFG,
				r.Style.FgMin,
				r.Style.FgMax,
				cursor(r.EstFG, r.Style.FgMin, r.Style.FgMax),
				(r.Style.FgMin <= r.EstFG && r.EstFG <= r.Style.FgMax),
				"",
			},
			"ABV": {
				r.EstABV,
				r.Style.AbvMin,
				r.Style.AbvMax,
				cursor(r.EstABV, r.Style.AbvMin, r.Style.AbvMax),
				(r.Style.AbvMin <= r.EstABV && r.EstABV <= r.Style.AbvMax),
				"",
			},
			"IBU": {
				r.IBU,
				r.Style.IbuMin,
				r.Style.IbuMax,
				cursor(r.IBU, r.Style.IbuMin, r.Style.IbuMax),
				(r.Style.IbuMin <= r.IBU && r.IBU <= r.Style.IbuMax),
				"",
			},
			"Color": {
				math.Round(r.EstColor * 100) / 100,
				r.Style.ColorMin,
				r.Style.ColorMax,
				cursor(r.EstColor, r.Style.ColorMin, r.Style.ColorMax),
				(r.Style.ColorMin <= r.EstColor && r.EstColor <= r.Style.ColorMax),
				template.CSS(fmt.Sprintf("rgb(%d, %d, %d)", hex.R, hex.G, hex.B)),
			},
			"IBU/OG": {
				math.Round(ibuOg * 100) / 100,
				0.2,
				1.2,
				cursor(ibuOg, 0.2, 1.2),
				true,
				"",
			},
			"IBU/RE": {
				math.Round(ibuRe * 100) / 100,
				0,
				15,
				cursor(ibuRe, 0, 15),
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

// Clone an existing recipe.
func (s *Server) cloneRecipe(w http.ResponseWriter, r *http.Request, user *db.User) {
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

	// Update name & version.
	recipe.Name = fmt.Sprintf("%s (cloned)", recipe.Name)
	recipe.XML.Name = recipe.Name
	recipe.XML.Version += 1;

	// Save or update recipe.
	clone, err := s.db.AddRecipe(recipe)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipe/%d", clone), 302)
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
		var fermentable beerxml.Fermentable
		if err := formToFermentable(r, &fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "add-hop":
		var hop beerxml.Hop
		if err := formToHop(r, &hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "add-yeast":
		var yeast beerxml.Yeast
		if err := formToYeast(r, &yeast); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &yeast); err != nil {
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
	case "edit-fermentable":
		var fermentable beerxml.Fermentable
		if err := formToFermentable(r, &fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Fermentable{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &fermentable); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "edit-hop":
		var hop beerxml.Hop
		if err := formToHop(r, &hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Hop{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &hop); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "edit-yeast":
		var yeast beerxml.Yeast
		if err := formToYeast(r, &yeast); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.RemoveFromRecipe(recipe.XML, &beerxml.Yeast{}, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := beerxml.InsertToRecipe(recipe.XML, &yeast); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "edit-mash-step":
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

	// Sort elements.
	sort(recipe.XML.Fermentables)
	sort(recipe.XML.Hops)
	sort(recipe.XML.Yeasts)
	sort(recipe.XML.Mash.MashSteps)

	// Update recipe.
	if err = s.db.UpdateRecipe(recipe); err != nil {
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
	recipe.XML.Name = recipe.Name
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

	recipe.XML.PrimaryAge, _ = strconv.ParseFloat(r.FormValue("primary-age"), 64)
	recipe.XML.PrimaryTemp, _ = strconv.ParseFloat(r.FormValue("primary-temp"), 64)
	recipe.XML.SecondaryAge, _ = strconv.ParseFloat(r.FormValue("secondary-age"), 64)
	recipe.XML.SecondaryTemp, _ = strconv.ParseFloat(r.FormValue("secondary-temp"), 64)
	recipe.XML.TertiaryAge, _ = strconv.ParseFloat(r.FormValue("tertiary-age"), 64)
	recipe.XML.TertiaryTemp, _ = strconv.ParseFloat(r.FormValue("tertiary-temp"), 64)
	recipe.XML.Age, _ = strconv.ParseFloat(r.FormValue("age"), 64)
	recipe.XML.AgeTemp, _ = strconv.ParseFloat(r.FormValue("age-temp"), 64)

	// Sanity checks.
	if recipe.Name == "" {
		return fmt.Errorf("'Name' is required.")
	}

	// Compute estimations.

	recipe.XML.EstOG = math.Round(recipe.XML.CalcOG() * 1000) / 1000
	recipe.XML.EstFG = math.Round(recipe.XML.CalcFG() * 1000) / 1000
	recipe.XML.EstABV = math.Round(recipe.XML.CalcABV() * 10) / 10
	recipe.XML.IBU = math.Round(recipe.XML.CalcIBU() * 10) / 10
	recipe.XML.EstColor = math.Round(recipe.XML.CalcColor() *  100) / 100

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
func formToFermentable(r *http.Request, fermentable *beerxml.Fermentable) error {
	fermentable.Name = r.FormValue("name")
	fermentable.Type = r.FormValue("type")
	fermentable.Yield, _ = strconv.ParseFloat(r.FormValue("yield"), 64)
	fermentable.Color, _ = strconv.ParseFloat(r.FormValue("color"), 64)
	fermentable.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)

	// Sanity checks
	if fermentable.Name == "" {
		return fmt.Errorf("'Name' is required.")
	}

	return nil
}

// Convert elements POSTed from a form into a beerxml.Hop.
func formToHop(r *http.Request, hop *beerxml.Hop) error {
	hop.Name = r.FormValue("name")
	hop.Form = r.FormValue("form")
	hop.Use = r.FormValue("use")
	hop.Alpha, _ = strconv.ParseFloat(r.FormValue("alpha"), 64)
	hop.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)
	hop.Time, _ = strconv.ParseFloat(r.FormValue("time"), 64)

	// Sanity checks
	if hop.Name == "" {
		return fmt.Errorf("'Name' is required.")
	}

	return nil
}

// Convert elements POSTed from a form into a beerxml.Yeast.
func formToYeast(r *http.Request, yeast *beerxml.Yeast) error {
	yeast.Name = r.FormValue("name")
	yeast.Form = r.FormValue("form")
	yeast.Attenuation, _ = strconv.ParseFloat(r.FormValue("attenuation"), 64)
	yeast.Amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)
	yeast.AmountIsWeight = r.FormValue("unit") == "kilogram"

	// Sanity checks
	if yeast.Name == "" {
		return fmt.Errorf("'Name' is required.")
	}

	return nil
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
