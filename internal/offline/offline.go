// Package offline provides the built-in solar system definition that lets the
// application run without any network access. The definition is embedded at
// build time and validated with the same rules used for online API data.
package offline

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/furan917/go-solar-system/internal/models"
)

// builtinSolarSystemFile is the canonical, curated solar system definition.
// It contains only the Sun, the eight planets and Pluto; no satellites.
//
//go:embed solar-system.json
var builtinSolarSystemFile []byte

// maxBuiltinBodies bounds the size of the embedded definition.
const maxBuiltinBodies = 100

// expectedBody describes one canonical record of the built-in definition.
type expectedBody struct {
	englishName string
	isPlanet    bool
	bodyType    string
}

// canonicalBodies fixes both the required members and their stable order:
// the Sun first, then the planets by distance from the Sun, ending with Pluto.
var canonicalBodies = []expectedBody{
	{englishName: "Sun", isPlanet: false, bodyType: "Star"},
	{englishName: "Mercury", isPlanet: true, bodyType: "Planet"},
	{englishName: "Venus", isPlanet: true, bodyType: "Planet"},
	{englishName: "Earth", isPlanet: true, bodyType: "Planet"},
	{englishName: "Mars", isPlanet: true, bodyType: "Planet"},
	{englishName: "Jupiter", isPlanet: true, bodyType: "Planet"},
	{englishName: "Saturn", isPlanet: true, bodyType: "Planet"},
	{englishName: "Uranus", isPlanet: true, bodyType: "Planet"},
	{englishName: "Neptune", isPlanet: true, bodyType: "Planet"},
	{englishName: "Pluto", isPlanet: false, bodyType: "Dwarf Planet"},
}

// LoadOfflineSolarSystem reads the embedded solar system definition, validates
// it completely and returns its bodies in canonical order. The returned slice
// uses the same models.CelestialBody type as the online API loader, so callers
// can consume both sources identically. A corrupted embedded file yields an
// error instead of a partially populated or different system.
func LoadOfflineSolarSystem() ([]models.CelestialBody, error) {
	return loadSolarSystem(builtinSolarSystemFile)
}

// loadSolarSystem parses and fully validates raw built-in definition bytes.
// It is split out from LoadOfflineSolarSystem so corrupted or malformed
// definitions can be exercised deterministically in tests.
func loadSolarSystem(data []byte) ([]models.CelestialBody, error) {
	var response models.APIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse built-in solar system definition: %w", err)
	}

	if response.Bodies == nil {
		return nil, fmt.Errorf("built-in solar system definition is missing the bodies field")
	}

	if len(response.Bodies) == 0 {
		return nil, fmt.Errorf("built-in solar system definition contains no bodies")
	}

	if len(response.Bodies) > maxBuiltinBodies {
		return nil, fmt.Errorf(
			"built-in solar system definition contains too many bodies: %d (max: %d)",
			len(response.Bodies),
			maxBuiltinBodies,
		)
	}

	seenIDs := make(map[string]bool, len(response.Bodies))
	for i, body := range response.Bodies {
		if err := models.ValidateCelestialBody(body); err != nil {
			return nil, fmt.Errorf("built-in solar system body at index %d is invalid: %w", i, err)
		}

		// The curated definition must never mix in satellites, neither as
		// standalone records nor as nested moon lists.
		if len(body.Moons) > 0 {
			return nil, fmt.Errorf(
				"built-in solar system body %s must not contain moons, got %d",
				body.EnglishName,
				len(body.Moons),
			)
		}
		if body.AroundPlanet != nil {
			return nil, fmt.Errorf(
				"built-in solar system body %s must not orbit another planet",
				body.EnglishName,
			)
		}

		identity := body.ID
		if identity == "" {
			identity = body.EnglishName
		}
		if seenIDs[identity] {
			return nil, fmt.Errorf("built-in solar system contains duplicate body id %q", identity)
		}
		seenIDs[identity] = true
	}

	if len(response.Bodies) != len(canonicalBodies) {
		return nil, fmt.Errorf(
			"built-in solar system must contain exactly %d bodies (Sun, eight planets and Pluto), got %d",
			len(canonicalBodies),
			len(response.Bodies),
		)
	}

	for i, expected := range canonicalBodies {
		body := response.Bodies[i]
		if body.EnglishName != expected.englishName {
			return nil, fmt.Errorf(
				"built-in solar system body at index %d is %q, want %q in canonical order",
				i,
				body.EnglishName,
				expected.englishName,
			)
		}
		if body.IsPlanet != expected.isPlanet {
			return nil, fmt.Errorf(
				"built-in solar system body %s has isPlanet=%v, want %v",
				expected.englishName,
				body.IsPlanet,
				expected.isPlanet,
			)
		}
		if body.BodyType != expected.bodyType {
			return nil, fmt.Errorf(
				"built-in solar system body %s has bodyType=%q, want %q",
				expected.englishName,
				body.BodyType,
				expected.bodyType,
			)
		}
	}

	bodies := make([]models.CelestialBody, len(response.Bodies))
	copy(bodies, response.Bodies)

	return bodies, nil
}
