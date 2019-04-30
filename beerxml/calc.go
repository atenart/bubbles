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
	"math"
)

const (
	LToGallon = 0.264172
	KgToPound = 2.20462
	KgToOunce = 35.274
	SrmToEbc  = 1.97
)

// Compute the volume needed in total.
func (r *Recipe) CalcVolumeTot() float64 {
	var maltAmount float64
	for _, f := range r.Fermentables {
		if f.Type != "Grain" {
			continue
		}
		maltAmount += f.Amount
	}


	// The boiling size should be the addition of:
	// - the target batch size
	// - the loss in the wort (here we use 1l / kg)
	// - the evaporation of water during the boiling
	return r.CalcBoilSize() + maltAmount * 1
}

// Compute the volume needed before boiling.
func (r *Recipe) CalcBoilSize() float64 {
	var evapRate float64 = 0.2
	if r.Equipment.EvapRate > 0 {
		evapRate = r.Equipment.EvapRate / 100
	}

	return r.BatchSize * (1 + r.BoilTime / 60 * evapRate)
}

// Compute the original gravity. The gravity is the density of a liquid compared
// to that of water (1).
// https://www.brassageamateur.com/wiki/index.php/Formules#Densit.C3.A9_pr.C3.A9-.C3.A9bullition_selon_grain
func (r *Recipe) CalcOG() float64 {
	// Compute the extract quantity
	var e float64
	for _, f := range r.Fermentables {
		e += f.Amount * f.Yield / 100
	}
	e *= r.Efficiency / 100

	if e == 0 {
		return 0
	}
	return (r.BatchSize - (e / 1.59) + e) / r.BatchSize
}

// Compute the final gravity.
// https://www.brassageamateur.com/wiki/index.php/Formules#Densit.C3.A9_finale_th.C3.A9orique
func (r *Recipe) CalcFG() float64 {
	if r.CalcOG() == 0 {
		return 0
	}

	var attenuation, amount float64
	for _, y := range r.Yeasts {
		attenuation += y.Attenuation / 100 * y.Amount
		amount += y.Amount
	}
	attenuation /= amount

	di := (r.CalcOG() * 1000) - 1000
	df := di * (1 - attenuation)

	return (df + 1000) / 1000
}

// Compute the color (SRM).
// https://www.brassageamateur.com/wiki/index.php/Formules#Couleur_de_la_bi.C3.A8re
func (r *Recipe) CalcColor() float64 {
	var mcu float64
	for _, f := range r.Fermentables {
		if f.Type != "Grain" {
			continue
		}

		mcu += f.Amount * f.Color
	}

	if mcu == 0 {
		return 0
	}
	return 2.9396 * math.Pow(4.23 * mcu / r.BatchSize, 0.6859)
}

// Compute the estimated alcohol by volume.
// https://www.brassageamateur.com/wiki/index.php/Formules#Taux_d.27alcool
func (r *Recipe) CalcABV() float64 {
	abv := 76.08 * (r.CalcOG() - r.CalcFG()) / (1.775 - r.CalcOG())
	abv *= r.CalcFG() / 0.794
	return abv
}

// Compute the bitterness using the Tinseth formula.
// https://www.brassageamateur.com/wiki/index.php/Formules#Amertume_objective_de_la_bi.C3.A8re
func (r *Recipe) CalcIBU() float64 {
	og := r.CalcOG()

	var sum float64
	for _, h := range r.Hops {
		if h.Use == "Dry Hop" || h.Use == "Aroma" {
			continue
		}

		ibu := 1.65 * math.Pow(0.000125, og- 1)
		ibu *= (1 - math.Pow(math.E, -0.04 * h.Time)) / 4.15
		ibu *= (h.Alpha / 100 * (h.Amount * 1000) * 1000) / r.BatchSize

		sum += ibu
	}

	return sum
}

// Compute the BU:GU ratio.
// https://www.brassageamateur.com/wiki/index.php/Formules#Amertume_subjective
func (r *Recipe) CalcIbuOg() float64 {
	if r.CalcOG() == 0 {
		return 0
	}
	return r.CalcIBU() / ((r.CalcOG() * 1000) - 1000)
}

// Compute the IBU:RE ratio.
// https://www.brassageamateur.com/wiki/index.php/Formules#Amertume_subjective
func (r *Recipe) CalcIbuRe() float64 {
	if r.CalcOG() == 0 || r.CalcFG() == 0 {
		return 0
	}

	og := ((r.CalcOG() * 1000) - 1000) / 4
	fg := ((r.CalcFG() * 1000) - 1000) / 4

	return r.CalcIBU() / ((0.1808 * og) + (0.8192 * fg))
}
