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
	s "sort"

	"github.com/atenart/bubbles/beerxml"
	"github.com/atenart/bubbles/db"
)

// Sort a slice of beerxml.Fermentable.
func sortFermentables(fermentables []beerxml.Fermentable) {
	s.Slice(fermentables, func(i, j int) bool {
		// Largest quantity first, or
		// lowest color first, or
		// 'name' in alphabetical order.
		if fermentables[i].Amount != fermentables[j].Amount {
			return fermentables[i].Amount > fermentables[j].Amount
		} else if fermentables[i].Color != fermentables[j].Color {
			return fermentables[i].Color < fermentables[j].Color
		}

		return fermentables[i].Name < fermentables[j].Name
	})
}

// Sort a slice of beerxml.Hop.
func sortHops(hops []beerxml.Hop) {
	s.Slice(hops, func(i, j int) bool {
		uses := map[string]int{
			"Mash": 0,
			"First wort": 1,
			"Boil": 2,
			"Aroma": 3,
			"Dry hop": 4,
		}

		// Smallest 'use' priority first, or
		// longest 'time' first, or
		// largest 'amount' first, or
		// largest 'alpha' first, or fallback to
		// 'name' in alphabetical order.
		if hops[i].Use != hops[j].Use {
			return uses[hops[i].Use] < uses[hops[j].Use]
		} else if hops[i].Time != hops[j].Time {
			return hops[i].Time > hops[j].Time
		} else if hops[i].Amount != hops[j].Amount {
			return hops[i].Amount > hops[j].Amount
		} else if hops[i].Alpha != hops[j].Alpha {
			return hops[i].Alpha > hops[j].Alpha
		}

		return hops[i].Name < hops[j].Name
	})
}

// Sort a slice of beerxml.Yeast.
func sortYeasts(yeasts []beerxml.Yeast) {
	s.Slice(yeasts, func(i, j int) bool {
		// Largest 'quantity' first, or
		// largest 'attenuation' first, or fallback to
		// 'name' in alphabetical order.
		if yeasts[i].Amount != yeasts[j].Amount {
			return yeasts[i].Amount > yeasts[j].Amount
		} else if yeasts[i].Attenuation != yeasts[j].Attenuation {
			return yeasts[i].Attenuation > yeasts[j].Attenuation
		}

		return yeasts[i].Name < yeasts[j].Name
	})
}

// Sort a slice of beerxml.MashStep.
func sortMashSteps(steps []beerxml.MashStep) {
	s.Slice(steps, func(i, j int) bool {
		// Lowest temperature first.
		return steps[i].StepTemp < steps[j].StepTemp
	})
}

// Sort a slice of beerxml.Ingredient.
func sortIngredients(ingredients []*db.Ingredient) {
	s.Slice(ingredients, func(i, j int) bool {
		uses := map[string]int{
			"fermentable": 0,
			"hop": 1,
			"yeast": 2,
		}

		// Smallest 'type' priority first, or fallback to
		// 'name' in alphabetical order.
		if ingredients[i].Type != ingredients[j].Type {
			return uses[ingredients[i].Type] < uses[ingredients[j].Type]
		}

		return ingredients[i].Name < ingredients[j].Name
	})
}

func sort(slice interface{}) {
	switch elmt := slice.(type) {
	case []beerxml.Fermentable:
		sortFermentables(elmt)
	case []beerxml.Hop:
		sortHops(elmt)
	case []beerxml.Yeast:
		sortYeasts(elmt)
	case []beerxml.MashStep:
		sortMashSteps(elmt)
	case []*db.Ingredient:
		sortIngredients(elmt)
	}
}
