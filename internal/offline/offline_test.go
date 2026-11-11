package offline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/furan917/go-solar-system/internal/models"
)

func canonicalNames() []string {
	names := make([]string, len(canonicalBodies))
	for i, body := range canonicalBodies {
		names[i] = body.englishName
	}
	return names
}

// validDefinitionBodies returns a fresh in-memory copy of the canonical
// definition so tests can mutate individual records safely.
func validDefinitionBodies() []models.CelestialBody {
	bodies := make([]models.CelestialBody, len(canonicalBodies))
	for i, expected := range canonicalBodies {
		bodies[i] = models.CelestialBody{
			ID:            strings.ToLower(expected.englishName),
			Name:          expected.englishName,
			EnglishName:   expected.englishName,
			IsPlanet:      expected.isPlanet,
			BodyType:      expected.bodyType,
			Moons:         []models.Moon{},
			MeanRadius:    1000 + float64(i),
			SemimajorAxis: float64(i) * 1000,
			Eccentricity:  0.01,
			Density:       1.0,
			Gravity:       1.0,
			Mass:          models.Mass{MassValue: 1, MassExponent: 20},
		}
	}
	return bodies
}

func marshalDefinition(t *testing.T, bodies []models.CelestialBody) []byte {
	t.Helper()

	data, err := json.Marshal(models.APIResponse{Bodies: bodies})
	if err != nil {
		t.Fatalf("marshal definition: %v", err)
	}

	return data
}

func TestLoadOfflineSolarSystemGoldenFile(t *testing.T) {
	bodies, err := LoadOfflineSolarSystem()
	if err != nil {
		t.Fatalf("LoadOfflineSolarSystem returned error: %v", err)
	}

	wantNames := canonicalNames()
	if len(bodies) != len(wantNames) {
		t.Fatalf("got %d bodies, want %d", len(bodies), len(wantNames))
	}

	for i, want := range wantNames {
		if bodies[i].EnglishName != want {
			t.Errorf("body at index %d = %q, want %q", i, bodies[i].EnglishName, want)
		}
	}

	if bodies[0].EnglishName != "Sun" {
		t.Errorf("first body must be the Sun, got %q", bodies[0].EnglishName)
	}
	if bodies[len(bodies)-1].EnglishName != "Pluto" {
		t.Errorf("last body must be Pluto, got %q", bodies[len(bodies)-1].EnglishName)
	}

	planetCount := 0
	for _, body := range bodies {
		if body.IsPlanet {
			planetCount++
		}
		if len(body.Moons) != 0 {
			t.Errorf("body %s must not contain moons, got %d", body.EnglishName, len(body.Moons))
		}
		if body.AroundPlanet != nil {
			t.Errorf("body %s must not orbit another planet", body.EnglishName)
		}
		if body.MeanRadius <= 0 {
			t.Errorf("body %s missing display field meanRadius", body.EnglishName)
		}
		if body.GetMassKg() <= 0 {
			t.Errorf("body %s missing mass", body.EnglishName)
		}
	}

	if planetCount != 8 {
		t.Errorf("got %d planets, want 8", planetCount)
	}

	pluto := bodies[len(bodies)-1]
	if pluto.IsPlanet {
		t.Error("Pluto must not be flagged as a planet")
	}
	if pluto.BodyType != "Dwarf Planet" {
		t.Errorf("Pluto bodyType = %q, want Dwarf Planet", pluto.BodyType)
	}

	// Orbit fields must be present for every orbiting body.
	for _, body := range bodies[1:] {
		if body.SemimajorAxis <= 0 {
			t.Errorf("body %s missing semimajorAxis", body.EnglishName)
		}
		if body.Perihelion <= 0 || body.Aphelion <= 0 {
			t.Errorf("body %s missing perihelion/aphelion", body.EnglishName)
		}
		if body.SideralOrbit <= 0 {
			t.Errorf("body %s missing sideralOrbit", body.EnglishName)
		}
	}
}

func TestLoadOfflineSolarSystemReturnsStableOrder(t *testing.T) {
	first, err := LoadOfflineSolarSystem()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	second, err := LoadOfflineSolarSystem()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("load counts differ: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].EnglishName != second[i].EnglishName || first[i].ID != second[i].ID {
			t.Fatalf("order unstable at index %d: %q/%q vs %q/%q",
				i, first[i].EnglishName, first[i].ID, second[i].EnglishName, second[i].ID)
		}
	}
}

func TestLoadSolarSystemRejectsCorruptJSON(t *testing.T) {
	if _, err := loadSolarSystem([]byte("{not valid json")); err == nil {
		t.Fatal("corrupt JSON must produce an error")
	}
}

func TestLoadSolarSystemRejectsMissingBodiesField(t *testing.T) {
	if _, err := loadSolarSystem([]byte(`{}`)); err == nil {
		t.Fatal("missing bodies field must produce an error")
	}
}

func TestLoadSolarSystemRejectsEmptyBodies(t *testing.T) {
	data, err := json.Marshal(models.APIResponse{Bodies: []models.CelestialBody{}})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := loadSolarSystem(data); err == nil {
		t.Fatal("empty bodies must produce an error")
	}
}

func TestLoadSolarSystemRejectsInvalidBody(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[2].MeanRadius = -10

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("negative mean radius must produce an error")
	}
}

func TestLoadSolarSystemRejectsUnrealisticEccentricity(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[2].Eccentricity = 1.5

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("eccentricity above 1 must produce an error")
	}
}

func TestLoadSolarSystemRejectsMoons(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[3].Moons = []models.Moon{{ID: "moon", EnglishName: "Moon"}}

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("a body containing moons must produce an error")
	}
}

func TestLoadSolarSystemRejectsSatelliteRecord(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[3].AroundPlanet = &models.Planet{ID: "sun", EnglishName: "Sun"}

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("a body orbiting another planet must produce an error")
	}
}

func TestLoadSolarSystemRejectsDuplicateID(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[1].ID = bodies[0].ID

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("duplicate body ids must produce an error")
	}
}

func TestLoadSolarSystemRejectsMissingRequiredBody(t *testing.T) {
	bodies := validDefinitionBodies()
	// Drop Pluto, the last record.
	if _, err := loadSolarSystem(marshalDefinition(t, bodies[:len(bodies)-1])); err == nil {
		t.Fatal("a definition missing Pluto must produce an error")
	}
}

func TestLoadSolarSystemRejectsExtraneousBody(t *testing.T) {
	bodies := append(validDefinitionBodies(), models.CelestialBody{
		ID:          "ceres",
		EnglishName: "Ceres",
		BodyType:    "Dwarf Planet",
		MeanRadius:  473,
	})

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("a definition with an extraneous body must produce an error")
	}
}

func TestLoadSolarSystemRejectsWrongOrder(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[1], bodies[2] = bodies[2], bodies[1]

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("bodies out of canonical order must produce an error")
	}
}

func TestLoadSolarSystemRejectsWrongTypeFlags(t *testing.T) {
	bodies := validDefinitionBodies()
	bodies[0].BodyType = "Planet"

	if _, err := loadSolarSystem(marshalDefinition(t, bodies)); err == nil {
		t.Fatal("a Sun mislabeled as Planet must produce an error")
	}
}
