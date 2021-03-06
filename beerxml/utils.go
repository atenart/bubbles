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

package beerxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

func Import(r io.Reader, data interface{}) error {
	d := xml.NewDecoder(r)
	return d.Decode(data)
}

// Open a Beer XML formated file and returns a BeerXML object.
func ImportFile(file string, data interface{}) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return Import(f, data)
}

func Export(data interface{}, w io.Writer) error {
	e := xml.NewEncoder(w)
	e.Indent("", "  ")
	return e.Encode(data)
}

// Write a BeerXML object to a file.
func ExportFile(data interface{}, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return Export(data, f)
}

// Insert an element into a BeerXML object.
func InsertToXML(xml *BeerXML, e interface{}) error {
	switch elmt := e.(type) {
	case *Recipe:
		xml.Recipes = append(xml.Recipes, *elmt)
	case *Fermentable:
		xml.Fermentables = append(xml.Fermentables, *elmt)
	case *Hop:
		xml.Hops = append(xml.Hops, *elmt)
	case *Yeast:
		xml.Yeasts = append(xml.Yeasts, *elmt)
	case *Misc:
		xml.Miscs = append(xml.Miscs, *elmt)
	case *Water:
		xml.Waters = append(xml.Waters, *elmt)
	default:
		return fmt.Errorf("Can't insert element, unknown type %T", elmt)
	}

	return nil
}

// Insert a recipe element into a Recipe object.
func InsertToRecipe(recipe *Recipe, e interface{}) error {
	switch elmt := e.(type) {
	case *Fermentable:
		recipe.Fermentables = append(recipe.Fermentables, *elmt)
	case *Hop:
		recipe.Hops = append(recipe.Hops, *elmt)
	case *Yeast:
		recipe.Yeasts = append(recipe.Yeasts, *elmt)
	case *Misc:
		recipe.Miscs = append(recipe.Miscs, *elmt)
	case *Water:
		recipe.Waters = append(recipe.Waters, *elmt)
	case *MashStep:
		recipe.Mash.MashSteps = append(recipe.Mash.MashSteps, *elmt)
	default:
		return fmt.Errorf("Can't insert element, unknown type %T", elmt)
	}

	return nil
}

// Remove a recipe element from a recipe object.
func RemoveFromRecipe(recipe *Recipe, e interface{}, key int) error {
	if key < 0 {
		return fmt.Errorf("Item not found.")
	}

	switch elmt := e.(type) {
	case *Hop:
		if key >= len(recipe.Hops) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Hops = append(recipe.Hops[:key], recipe.Hops[key+1:]...)
	case *Fermentable:
		if key >= len(recipe.Fermentables) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Fermentables = append(recipe.Fermentables[:key], recipe.Fermentables[key+1:]...)
	case *Yeast:
		if key >= len(recipe.Yeasts) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Yeasts = append(recipe.Yeasts[:key], recipe.Yeasts[key+1:]...)
	case *Misc:
		if key >= len(recipe.Miscs) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Miscs = append(recipe.Miscs[:key], recipe.Miscs[key+1:]...)
	case *Water:
		if key >= len(recipe.Waters) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Waters = append(recipe.Waters[:key], recipe.Waters[key+1:]...)
	case *MashStep:
		if key >= len(recipe.Mash.MashSteps) {
			return fmt.Errorf("Item not found.")
		}
		recipe.Mash.MashSteps = append(recipe.Mash.MashSteps[:key], recipe.Mash.MashSteps[key+1:]...)
	default:
		return fmt.Errorf("Can't insert element, unknown type %T", elmt)
	}

	return nil
}
