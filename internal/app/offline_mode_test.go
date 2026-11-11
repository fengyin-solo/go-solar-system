package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/furan917/go-solar-system/internal/api"
	"github.com/furan917/go-solar-system/internal/models"
	"github.com/furan917/go-solar-system/internal/systems"
	"github.com/furan917/go-solar-system/internal/visualization"
	"github.com/gdamore/tcell/v2"
)

func offlineFixtureBodies() []models.CelestialBody {
	return []models.CelestialBody{
		{
			ID: "sun", EnglishName: "Sun", Name: "Sun", IsPlanet: false,
			BodyType: "Star", MeanRadius: 695700, SemimajorAxis: 0,
		},
		{
			ID: "terre", EnglishName: "Earth", Name: "Terre", IsPlanet: true,
			BodyType: "Planet", MeanRadius: 6371, SemimajorAxis: 149597890,
		},
		{
			ID: "mars", EnglishName: "Mars", Name: "Mars", IsPlanet: true,
			BodyType: "Planet", MeanRadius: 3389.5, SemimajorAxis: 227939200,
		},
	}
}

func newTestSystemsManager(t *testing.T) *systems.SystemManager {
	t.Helper()

	bundled := fstest.MapFS{
		"alpha-centauri.json": {Data: []byte(testSystemJSON("Alpha Centauri", "Closest binary system"))},
	}

	sm := systems.NewSystemManager(bundled)
	if err := sm.ScanSystems(); err != nil {
		t.Fatalf("ScanSystems: %v", err)
	}

	return sm
}

func newTestPlanetService(t *testing.T, client *api.Client, opts ...PlanetServiceOption) (*PlanetService, *systems.SystemManager, *bytes.Buffer) {
	t.Helper()

	systemManager := newTestSystemsManager(t)
	logBuffer := &bytes.Buffer{}
	allOpts := append([]PlanetServiceOption{WithPlanetServiceLogger(log.New(logBuffer, "", 0))}, opts...)

	return NewPlanetService(client, systemManager, allOpts...), systemManager, logBuffer
}

func bodyNames(bodies []models.CelestialBody) []string {
	names := make([]string, len(bodies))
	for i, body := range bodies {
		names[i] = body.EnglishName
	}
	return names
}

func TestPlanetServiceExplicitOfflineSkipsNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("offline mode must not contact the API, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	loaderCalls := 0
	loader := func() ([]models.CelestialBody, error) {
		loaderCalls++
		return offlineFixtureBodies(), nil
	}

	client := api.NewClientWithBaseURL(server.URL)
	ps, systemManager, _ := newTestPlanetService(t, client, WithOfflineMode(), WithOfflineLoader(loader))

	if !ps.IsOffline() {
		t.Fatal("service should report offline mode")
	}

	bodies, err := ps.LoadSolarSystemBodies()
	if err != nil {
		t.Fatalf("LoadSolarSystemBodies: %v", err)
	}

	if loaderCalls != 1 {
		t.Errorf("offline loader calls = %d, want 1", loaderCalls)
	}
	if got := bodyNames(bodies); got[0] != "Sun" || got[1] != "Earth" || got[2] != "Mars" {
		t.Errorf("offline bodies = %v, want fixture order", got)
	}
	if got := systemManager.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("current system = %q, want solar-system", got)
	}
}

func TestPlanetServiceOnlineSuccessUsesAPIWithoutWarning(t *testing.T) {
	response := models.APIResponse{Bodies: []models.CelestialBody{
		{ID: "terre", EnglishName: "Earth", IsPlanet: true, MeanRadius: 6371, SemimajorAxis: 149597890},
		{ID: "mars", EnglishName: "Mars", IsPlanet: true, MeanRadius: 3389.5, SemimajorAxis: 227939200},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ps, _, logBuffer := newTestPlanetService(t, api.NewClientWithBaseURL(server.URL))

	bodies, err := ps.LoadSolarSystemBodies()
	if err != nil {
		t.Fatalf("LoadSolarSystemBodies: %v", err)
	}

	if got := bodyNames(bodies); len(got) != 2 || got[0] != "Earth" || got[1] != "Mars" {
		t.Errorf("online bodies = %v, want [Earth Mars]", got)
	}
	if logBuffer.Len() != 0 {
		t.Errorf("successful online load must not emit warnings, got %q", logBuffer.String())
	}
}

func TestPlanetServiceOnlineFailureFallsBackWithWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ps, systemManager, logBuffer := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return offlineFixtureBodies(), nil
		}),
	)

	bodies, err := ps.LoadSolarSystemBodies()
	if err != nil {
		t.Fatalf("fallback should succeed, got error: %v", err)
	}

	if got := bodyNames(bodies); len(got) != 3 || got[0] != "Sun" {
		t.Errorf("fallback bodies = %v, want built-in fixture", got)
	}
	if got := systemManager.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("fallback must keep solar-system, current = %q", got)
	}

	warning := logBuffer.String()
	if !strings.Contains(warning, "WARNING") || !strings.Contains(warning, "built-in offline solar system") {
		t.Errorf("stderr warning should explain the fallback, got %q", warning)
	}
}

func TestPlanetServiceOnlineEmptyPlanetsFallsBack(t *testing.T) {
	// Valid body, but nothing flagged as a planet: GetPlanets returns an empty slice.
	response := models.APIResponse{Bodies: []models.CelestialBody{
		{ID: "lune", EnglishName: "Moon", IsPlanet: false, MeanRadius: 1737},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ps, _, logBuffer := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return offlineFixtureBodies(), nil
		}),
	)

	bodies, err := ps.LoadSolarSystemBodies()
	if err != nil {
		t.Fatalf("empty planet response should trigger fallback, got error: %v", err)
	}
	if len(bodies) != 3 {
		t.Errorf("expected fallback fixture, got %d bodies", len(bodies))
	}
	if !strings.Contains(logBuffer.String(), "WARNING") {
		t.Errorf("empty response should log a fallback warning, got %q", logBuffer.String())
	}
}

func TestPlanetServiceFallbackFailureReturnsErrorWithoutSwitching(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	loaderErr := errors.New("built-in definition corrupted")
	ps, systemManager, _ := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return nil, loaderErr
		}),
	)

	if _, err := ps.LoadSolarSystemBodies(); err == nil {
		t.Fatal("a failed fallback must return an error instead of silent wrong data")
	}
	if got := systemManager.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("failure must not jump to another system, current = %q", got)
	}
}

func TestPlanetServiceExplicitOfflineLoaderFailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("offline mode must not contact the API")
	}))
	defer server.Close()

	ps, _, _ := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineMode(),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return nil, errors.New("built-in definition corrupted")
		}),
	)

	if _, err := ps.LoadSolarSystemBodies(); err == nil {
		t.Fatal("corrupted built-in data in offline mode must fail")
	}
}

func TestPlanetServiceExternalSystemSwitchingStillWorks(t *testing.T) {
	response := models.APIResponse{Bodies: []models.CelestialBody{
		{ID: "terre", EnglishName: "Earth", IsPlanet: true, MeanRadius: 6371, SemimajorAxis: 1},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	systemManager := newTestSystemsManager(t)
	logger := log.New(io.Discard, "", 0)

	online := NewPlanetService(api.NewClientWithBaseURL(server.URL), systemManager,
		WithPlanetServiceLogger(logger))
	offline := NewPlanetService(api.NewClientWithBaseURL(server.URL), systemManager,
		WithPlanetServiceLogger(logger),
		WithOfflineMode(),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return offlineFixtureBodies(), nil
		}),
	)

	external, err := online.SwitchToSystem("alpha-centauri")
	if err != nil {
		t.Fatalf("SwitchToSystem(alpha-centauri): %v", err)
	}
	if got := systemManager.GetCurrentSystem(); got != "alpha-centauri" {
		t.Fatalf("current system = %q, want alpha-centauri", got)
	}
	if len(external) != 1 || external[0].EnglishName != "star" {
		t.Errorf("external bodies = %+v, want the file-defined body", external)
	}

	// Reloading the external system through LoadCurrentSystem stays file-based.
	reloaded, err := online.LoadCurrentSystem()
	if err != nil {
		t.Fatalf("LoadCurrentSystem for external system: %v", err)
	}
	if len(reloaded) != 1 {
		t.Errorf("external reload returned %d bodies, want 1", len(reloaded))
	}

	// Switching back to the solar system honors explicit offline mode.
	solarPlanets, err := offline.SwitchToSystem("solar-system")
	if err != nil {
		t.Fatalf("SwitchToSystem(solar-system) in offline mode: %v", err)
	}
	if got := systemManager.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("current system = %q, want solar-system", got)
	}
	// The planet-only view keeps the historical IsPlanet filter contract.
	if got := bodyNames(solarPlanets); len(got) != 2 || got[0] != "Earth" || got[1] != "Mars" {
		t.Errorf("offline planet view after switch back = %v, want [Earth Mars]", got)
	}

	// The full body view used at startup includes the Sun and stays offline.
	solarBodies, err := offline.LoadSolarSystemBodies()
	if err != nil {
		t.Fatalf("LoadSolarSystemBodies after switch back: %v", err)
	}
	if got := bodyNames(solarBodies); len(got) != 3 || got[0] != "Sun" {
		t.Errorf("offline full bodies after switch back = %v, want fixture with Sun", got)
	}
}

func newAppSystemManager(t *testing.T, ps *PlanetService, sm *systems.SystemManager) (*SystemManager, *AppState) {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(140, 40)

	state := NewAppState()
	logger := log.New(io.Discard, "", 0)
	uiRenderer := NewUIRenderer(screen, visualization.NewRendererWithDefaults(140, 40), sm, state)
	errorHandler := NewErrorHandler(logger, state)

	return NewSystemManager(state, ps, uiRenderer, errorHandler, logger), state
}

func TestAppLoadCurrentSystemOfflineKeepsSolarSystem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("offline mode must not contact the API")
	}))
	defer server.Close()

	ps, sm, _ := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineMode(),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return offlineFixtureBodies(), nil
		}),
	)

	appSM, state := newAppSystemManager(t, ps, sm)
	if err := appSM.LoadCurrentSystem(); err != nil {
		t.Fatalf("LoadCurrentSystem: %v", err)
	}

	if got := sm.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("must stay on solar-system, got %q", got)
	}
	if got := bodyNames(state.GetPlanets()); len(got) != 3 || got[0] != "Sun" {
		t.Errorf("state bodies = %v, want offline fixture", got)
	}
}

func TestAppLoadCurrentSystemOnlineFailureUsesBuiltinFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	// Uses the real embedded built-in definition via the default loader.
	ps, sm, _ := newTestPlanetService(t, api.NewClientWithBaseURL(server.URL))

	appSM, state := newAppSystemManager(t, ps, sm)
	if err := appSM.LoadCurrentSystem(); err != nil {
		t.Fatalf("LoadCurrentSystem should fall back to built-in data: %v", err)
	}

	if got := sm.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("fallback must keep solar-system, got %q", got)
	}

	names := bodyNames(state.GetPlanets())
	if len(names) != 10 {
		t.Fatalf("built-in fallback bodies = %d, want 10", len(names))
	}
	if names[0] != "Sun" || names[1] != "Mercury" || names[len(names)-1] != "Pluto" {
		t.Errorf("built-in fallback order wrong: %v", names)
	}
}

func TestAppLoadCurrentSystemFallbackFailureAborts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ps, sm, _ := newTestPlanetService(t,
		api.NewClientWithBaseURL(server.URL),
		WithOfflineLoader(func() ([]models.CelestialBody, error) {
			return nil, errors.New("built-in definition corrupted")
		}),
	)

	appSM, _ := newAppSystemManager(t, ps, sm)
	if err := appSM.LoadCurrentSystem(); err == nil {
		t.Fatal("corrupted built-in fallback must abort startup with an error")
	}

	if got := sm.GetCurrentSystem(); got != "solar-system" {
		t.Errorf("failure must never silently switch systems, got %q", got)
	}
}
