package models

import (
	"fmt"
	"math"
	"time"
)

type CelestialBody struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	EnglishName     string  `json:"englishName"`
	IsPlanet        bool    `json:"isPlanet"`
	Moons           []Moon  `json:"moons"`
	SemimajorAxis   float64 `json:"semimajorAxis"`
	Perihelion      float64 `json:"perihelion"`
	Aphelion        float64 `json:"aphelion"`
	Eccentricity    float64 `json:"eccentricity"`
	Inclination     float64 `json:"inclination"`
	Mass            Mass    `json:"mass"`
	Vol             Vol     `json:"vol"`
	Density         float64 `json:"density"`
	Gravity         float64 `json:"gravity"`
	Escape          float64 `json:"escape"`
	MeanRadius      float64 `json:"meanRadius"`
	EquaRadius      float64 `json:"equaRadius"`
	PolarRadius     float64 `json:"polarRadius"`
	Flattening      float64 `json:"flattening"`
	Dimension       string  `json:"dimension"`
	SideralOrbit    float64 `json:"sideralOrbit"`
	SideralRotation float64 `json:"sideralRotation"`
	AroundPlanet    *Planet `json:"aroundPlanet"`
	DiscoveredBy    string  `json:"discoveredBy"`
	DiscoveryDate   string  `json:"discoveryDate"`
	AlternativeName string  `json:"alternativeName"`
	BodyType        string  `json:"bodyType"`
	Rel             string  `json:"rel"`

	// Stellar properties
	Temperature  float64 `json:"temperature"`
	StellarClass string  `json:"stellarClass"`
	Age          float64 `json:"age"`

	// Orbital elements for precise positioning (optional)
	OrbitalElements *OrbitalElement `json:"orbitalElements,omitempty"`
}

type Planet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EnglishName string `json:"englishName"`
	Rel         string `json:"rel"`
}

type Moon struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EnglishName string `json:"englishName"`
	Rel         string `json:"rel"`
}

type Mass struct {
	MassValue    float64 `json:"massValue"`
	MassExponent int     `json:"massExponent"`
}

type Vol struct {
	VolValue    float64 `json:"volValue"`
	VolExponent int     `json:"volExponent"`
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
	SemimajorAxis            float64
	Eccentricity             float64
	Inclination              float64
	ArgumentOfPeriapsis      float64
	LongitudeOfAscendingNode float64
	MeanAnomaly              float64
	Epoch                    time.Time
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

// ValidateCelestialBody performs the same structural validation applied to
// bodies returned by the online API, so offline and online data are checked
// against identical rules.
func ValidateCelestialBody(body CelestialBody) error {
	// Must have at least one name identifier
	if body.EnglishName == "" && body.Name == "" && body.ID == "" {
		return fmt.Errorf("celestial body missing all name identifiers")
	}

	displayName := body.EnglishName
	if displayName == "" {
		displayName = body.Name
	}
	if displayName == "" {
		displayName = body.ID
	}

	if body.MeanRadius < 0 {
		return fmt.Errorf("celestial body %s has negative mean radius", displayName)
	}

	if body.SemimajorAxis < 0 {
		return fmt.Errorf("celestial body %s has negative semimajor axis", displayName)
	}

	if body.Density < 0 {
		return fmt.Errorf("celestial body %s has negative density", displayName)
	}

	if body.Gravity < 0 {
		return fmt.Errorf("celestial body %s has negative gravity", displayName)
	}

	if body.Eccentricity < 0 || body.Eccentricity > 1 {
		return fmt.Errorf(
			"celestial body %s has unrealistic eccentricity: %.6f",
			displayName,
			body.Eccentricity,
		)
	}

	return nil
}
