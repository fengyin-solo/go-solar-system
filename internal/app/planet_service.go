package app

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/furan917/go-solar-system/internal/api"
	"github.com/furan917/go-solar-system/internal/models"
	"github.com/furan917/go-solar-system/internal/offline"
	"github.com/furan917/go-solar-system/internal/systems"
)

// PlanetService handles business logic for celestial body operations
type PlanetService struct {
	client        *api.Client
	systemManager *systems.SystemManager

	// offline skips every network request and only uses built-in data.
	offline bool
	// offlineLoader loads the built-in solar system; it is a field so tests
	// can inject deterministic fixtures and failure cases.
	offlineLoader func() ([]models.CelestialBody, error)
	// logger receives fallback warnings and defaults to stderr.
	logger *log.Logger
}

// PlanetServiceOption customizes a PlanetService.
type PlanetServiceOption func(*PlanetService)

// WithOfflineMode forces the service to use the built-in solar system
// definition without ever contacting the remote API.
func WithOfflineMode() PlanetServiceOption {
	return func(ps *PlanetService) {
		ps.offline = true
	}
}

// WithOfflineLoader replaces the built-in loader. It is intended for tests
// that need deterministic offline data or need to simulate a corrupted
// built-in definition.
func WithOfflineLoader(loader func() ([]models.CelestialBody, error)) PlanetServiceOption {
	return func(ps *PlanetService) {
		if loader != nil {
			ps.offlineLoader = loader
		}
	}
}

// WithPlanetServiceLogger redirects fallback warnings to the given logger.
// Defaults to stderr when not provided.
func WithPlanetServiceLogger(logger *log.Logger) PlanetServiceOption {
	return func(ps *PlanetService) {
		if logger != nil {
			ps.logger = logger
		}
	}
}

// NewPlanetService creates a new planet service with necessary dependencies
func NewPlanetService(client *api.Client, systemManager *systems.SystemManager, opts ...PlanetServiceOption) *PlanetService {
	ps := &PlanetService{
		client:        client,
		systemManager: systemManager,
		offlineLoader: offline.LoadOfflineSolarSystem,
		logger:        log.New(os.Stderr, "", log.LstdFlags),
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

// IsOffline reports whether the service runs in explicit offline mode.
func (ps *PlanetService) IsOffline() bool {
	return ps.offline
}

// LoadCurrentSystem loads celestial bodies for the current system
func (ps *PlanetService) LoadCurrentSystem() ([]models.CelestialBody, error) {
	currentSystem := ps.systemManager.GetCurrentSystem()

	if currentSystem == "solar-system" {
		return ps.loadSolarSystem()
	}

	return ps.loadExternalSystem(currentSystem)
}

// LoadSolarSystemBodies returns the bodies displayed for our solar system.
//
// In explicit offline mode the remote API is never contacted: only the
// built-in definition is loaded and validated, and a failure there is
// returned to the caller so startup aborts instead of showing wrong data.
//
// In the default online mode the API is tried first. When the request fails
// or returns no planets, the built-in definition is used as fallback and a
// clear warning is written to stderr. A successful API request keeps the
// existing behavior and produces no extra output.
func (ps *PlanetService) LoadSolarSystemBodies() ([]models.CelestialBody, error) {
	if ps.offline {
		bodies, err := ps.offlineLoader()
		if err != nil {
			return nil, fmt.Errorf("failed to load built-in offline solar system: %w", err)
		}

		return bodies, nil
	}

	planets, err := ps.client.GetPlanets()
	if err == nil && len(planets) > 0 {
		return planets, nil
	}

	if err == nil {
		err = fmt.Errorf("API returned no planets")
	}

	ps.logger.Printf(
		"WARNING: Solar System API request failed (%v); using built-in offline solar system data instead",
		err,
	)

	bodies, fallbackErr := ps.offlineLoader()
	if fallbackErr != nil {
		return nil, fmt.Errorf(
			"solar system API unavailable (%v) and built-in offline fallback failed: %w",
			err,
			fallbackErr,
		)
	}

	return bodies, nil
}

// loadSolarSystem loads our solar system, honoring offline mode and the
// online-failure fallback, and returns its planets sorted by distance.
func (ps *PlanetService) loadSolarSystem() ([]models.CelestialBody, error) {
	bodies, err := ps.LoadSolarSystemBodies()
	if err != nil {
		return nil, fmt.Errorf("failed to load solar system: %w", err)
	}

	var planets []models.CelestialBody
	for _, body := range bodies {
		if body.IsPlanet {
			planets = append(planets, body)
		}
	}

	sort.Slice(planets, func(i, j int) bool {
		return planets[i].SemimajorAxis < planets[j].SemimajorAxis
	})

	return planets, nil
}

// loadExternalSystem loads an external star system from JSON files
func (ps *PlanetService) loadExternalSystem(systemName string) ([]models.CelestialBody, error) {
	systemData, err := ps.systemManager.LoadSystem(systemName)
	if err != nil {
		return nil, fmt.Errorf("failed to load external system %s: %w", systemName, err)
	}

	planets := systemData.Bodies
	sort.Slice(planets, func(i, j int) bool {
		return planets[i].SemimajorAxis < planets[j].SemimajorAxis
	})

	return planets, nil
}

// SwitchToSystem changes the current system and loads its data
func (ps *PlanetService) SwitchToSystem(systemName string) ([]models.CelestialBody, error) {
	if err := ps.systemManager.SwitchToSystem(systemName); err != nil {
		return nil, fmt.Errorf("failed to switch to system %s: %w", systemName, err)
	}

	return ps.LoadCurrentSystem()
}

// GetMoonData attempts to fetch detailed moon data
func (ps *PlanetService) GetMoonData(moonID string) (*models.CelestialBody, error) {
	return ps.client.GetMoonData(moonID)
}

// ValidatePlanetData performs basic validation on planet data
func (ps *PlanetService) ValidatePlanetData(planets []models.CelestialBody) error {
	if len(planets) == 0 {
		return fmt.Errorf("no planets loaded")
	}

	for i, planet := range planets {
		if planet.EnglishName == "" {
			return fmt.Errorf("planet at index %d has no name", i)
		}

		if planet.MeanRadius < 0 {
			return fmt.Errorf("planet %s has invalid radius: %.2f", planet.EnglishName, planet.MeanRadius)
		}
	}

	return nil
}

// GetClient returns the API client
func (ps *PlanetService) GetClient() *api.Client {
	return ps.client
}
