package models

import (
	"math"
	"time"
)

type CelestialBody struct {
	ID              string  `json:"id" yaml:"id"`
	Name            string  `json:"name" yaml:"name"`
	EnglishName     string  `json:"englishName" yaml:"englishName"`
	IsPlanet        bool    `json:"isPlanet" yaml:"isPlanet"`
	Moons           []Moon  `json:"moons" yaml:"moons"`
	SemimajorAxis   float64 `json:"semimajorAxis" yaml:"semimajorAxis"`
	Perihelion      float64 `json:"perihelion" yaml:"perihelion"`
	Aphelion        float64 `json:"aphelion" yaml:"aphelion"`
	Eccentricity    float64 `json:"eccentricity" yaml:"eccentricity"`
	Inclination     float64 `json:"inclination" yaml:"inclination"`
	Mass            Mass    `json:"mass" yaml:"mass"`
	Vol             Vol     `json:"vol" yaml:"vol"`
	Density         float64 `json:"density" yaml:"density"`
	Gravity         float64 `json:"gravity" yaml:"gravity"`
	Escape          float64 `json:"escape" yaml:"escape"`
	MeanRadius      float64 `json:"meanRadius" yaml:"meanRadius"`
	EquaRadius      float64 `json:"equaRadius" yaml:"equaRadius"`
	PolarRadius     float64 `json:"polarRadius" yaml:"polarRadius"`
	Flattening      float64 `json:"flattening" yaml:"flattening"`
	Dimension       string  `json:"dimension" yaml:"dimension"`
	SideralOrbit    float64 `json:"sideralOrbit" yaml:"sideralOrbit"`
	SideralRotation float64 `json:"sideralRotation" yaml:"sideralRotation"`
	AroundPlanet    *Planet `json:"aroundPlanet" yaml:"aroundPlanet"`
	DiscoveredBy    string  `json:"discoveredBy" yaml:"discoveredBy"`
	DiscoveryDate   string  `json:"discoveryDate" yaml:"discoveryDate"`
	AlternativeName string  `json:"alternativeName" yaml:"alternativeName"`
	BodyType        string  `json:"bodyType" yaml:"bodyType"`
	Rel             string  `json:"rel" yaml:"rel"`

	// Stellar properties
	Temperature  float64 `json:"temperature" yaml:"temperature"`
	StellarClass string  `json:"stellarClass" yaml:"stellarClass"`
	Age          float64 `json:"age" yaml:"age"`

	// Orbital elements for precise positioning (optional)
	OrbitalElements *OrbitalElement `json:"orbitalElements,omitempty" yaml:"orbitalElements,omitempty"`
}

type Planet struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	EnglishName string `json:"englishName" yaml:"englishName"`
	Rel         string `json:"rel" yaml:"rel"`
}

type Moon struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	EnglishName string `json:"englishName" yaml:"englishName"`
	Rel         string `json:"rel" yaml:"rel"`
}

type Mass struct {
	MassValue    float64 `json:"massValue" yaml:"massValue"`
	MassExponent int     `json:"massExponent" yaml:"massExponent"`
}

type Vol struct {
	VolValue    float64 `json:"volValue" yaml:"volValue"`
	VolExponent int     `json:"volExponent" yaml:"volExponent"`
}

type APIResponse struct {
	Bodies []CelestialBody `json:"bodies"`
}

type Position struct {
	X float64
	Y float64
	Z float64
}

type OrbitalElement struct {
	SemimajorAxis            float64   `yaml:"semimajorAxis"`
	Eccentricity             float64   `yaml:"eccentricity"`
	Inclination              float64   `yaml:"inclination"`
	ArgumentOfPeriapsis      float64   `yaml:"argumentOfPeriapsis"`
	LongitudeOfAscendingNode float64   `yaml:"longitudeOfAscendingNode"`
	MeanAnomaly              float64   `yaml:"meanAnomaly"`
	Epoch                    time.Time `yaml:"epoch"`
}

func (cb *CelestialBody) GetMassKg() float64 {
	if cb.Mass.MassValue == 0 {
		return 0
	}
	return cb.Mass.MassValue * math.Pow10(cb.Mass.MassExponent)
}

func (cb *CelestialBody) GetVolumeKm3() float64 {
	if cb.Vol.VolValue == 0 {
		return 0
	}
	return cb.Vol.VolValue * math.Pow10(cb.Vol.VolExponent)
}
