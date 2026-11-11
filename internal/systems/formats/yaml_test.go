package formats

import (
	"reflect"
	"strings"
	"testing"
)

const validSingleStarYAML = `
systemName: Solo System
description: A single-star system
discoveryYear: "2020"
distance: 12 light-years
galaxy: Milky Way
bodies:
  - id: solo-star
    name: Solo
    englishName: Solo Star
    bodyType: Star
    isPlanet: false
    meanRadius: 700000
    mass:
      massValue: 1.0
      massExponent: 30
    vol:
      volValue: 1.4
      volExponent: 27
    temperature: 5800
    stellarClass: G2V
    age: 5000000000
    moons:
      - id: solo-moon-1
        name: Luna
        englishName: Solo Moon
        rel: /bodies/solo-moon-1
    aroundPlanet:
      id: solo-planet
      name: Planet
      englishName: Solo Planet
      rel: /bodies/solo-planet
    orbitalElements:
      semimajorAxis: 149.6
      eccentricity: 0.017
      inclination: 7.15
      argumentOfPeriapsis: 102.9
      longitudeOfAscendingNode: 0.0
      meanAnomaly: 358.6
      epoch: "2020-01-01T00:00:00Z"
`

const validMultiStarYAML = `
systemName: Triple Stars
description: Three stars sharing one system
discoveryYear: "1850"
distance: 25 light-years
galaxy: Andromeda
bodies:
  - englishName: Star A
    bodyType: Star
    isPlanet: false
  - englishName: Star B
    bodyType: Star
    isPlanet: false
    moons:
      - englishName: Moon of B
  - englishName: Planet C
    bodyType: Planet
    isPlanet: true
    meanRadius: 6371
`

func requireContains(t *testing.T, text string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("error message %q does not contain %q", text, want)
		}
	}
}

func TestYAMLFormatMetadata(t *testing.T) {
	yf := NewYAMLFormat()

	if got := yf.GetFormatName(); got != "YAML" {
		t.Errorf("GetFormatName = %q, want YAML", got)
	}

	if got := yf.GetMimeType(); got != "application/x-yaml" {
		t.Errorf("GetMimeType = %q, want application/x-yaml", got)
	}

	exts := yf.GetSupportedExtensions()
	if len(exts) != 2 || exts[0] != ".yaml" || exts[1] != ".yml" {
		t.Errorf("GetSupportedExtensions = %v, want [.yaml .yml]", exts)
	}
}

func TestRegistryRegistersYAMLExtensions(t *testing.T) {
	registry := NewFormatRegistry()

	for _, ext := range []string{".yaml", ".yml"} {
		handler, ok := registry.GetHandlerForExtension(ext)
		if !ok {
			t.Fatalf("extension %s has no registered handler", ext)
		}
		if handler.GetFormatName() != "YAML" {
			t.Errorf("extension %s resolved to %s, want YAML", ext, handler.GetFormatName())
		}
	}

	if handler, ok := registry.GetHandlerForExtension(".json"); !ok || handler.GetFormatName() != "JSON" {
		t.Fatalf(".json should still resolve to JSON, got %v", handler)
	}

	// JSON must be registered first so JSON content is detected before YAML.
	formats := registry.GetAllFormats()
	if len(formats) < 2 || formats[0].GetFormatName() != "JSON" || formats[1].GetFormatName() != "YAML" {
		t.Fatalf("registration order = %v, want JSON before YAML", formats)
	}
}

func TestYAMLParseSingleStarSystem(t *testing.T) {
	yf := NewYAMLFormat()

	system, err := yf.ParseSystemData([]byte(validSingleStarYAML))
	if err != nil {
		t.Fatalf("ParseSystemData: %v", err)
	}

	if system.SystemName != "Solo System" {
		t.Errorf("SystemName = %q", system.SystemName)
	}
	if system.Description != "A single-star system" || system.DiscoveryYear != "2020" ||
		system.Distance != "12 light-years" || system.Galaxy != "Milky Way" {
		t.Errorf("metadata fields parsed incorrectly: %+v", system)
	}

	if len(system.Bodies) != 1 {
		t.Fatalf("expected 1 body, got %d", len(system.Bodies))
	}

	body := system.Bodies[0]
	if body.EnglishName != "Solo Star" || body.BodyType != "Star" || body.IsPlanet {
		t.Errorf("base body fields wrong: %+v", body)
	}
	if body.MeanRadius != 700000 || body.Temperature != 5800 || body.StellarClass != "G2V" || body.Age != 5000000000 {
		t.Errorf("stellar fields wrong: %+v", body)
	}
	if body.Mass.MassValue != 1.0 || body.Mass.MassExponent != 30 {
		t.Errorf("nested mass wrong: %+v", body.Mass)
	}
	if body.Vol.VolValue != 1.4 || body.Vol.VolExponent != 27 {
		t.Errorf("nested vol wrong: %+v", body.Vol)
	}
	if body.OrbitalElements == nil || body.OrbitalElements.SemimajorAxis != 149.6 ||
		body.OrbitalElements.Eccentricity != 0.017 || body.OrbitalElements.Inclination != 7.15 {
		t.Errorf("orbital elements wrong: %+v", body.OrbitalElements)
	}
	if body.OrbitalElements.Epoch.IsZero() {
		t.Error("orbital element epoch should have parsed as a timestamp")
	}
	if body.AroundPlanet == nil || body.AroundPlanet.EnglishName != "Solo Planet" {
		t.Errorf("nested aroundPlanet wrong: %+v", body.AroundPlanet)
	}
}

func TestYAMLParseMultiStarSystemAndMoons(t *testing.T) {
	yf := NewYAMLFormat()

	system, err := yf.ParseSystemData([]byte(validMultiStarYAML))
	if err != nil {
		t.Fatalf("ParseSystemData: %v", err)
	}

	if len(system.Bodies) != 3 {
		t.Fatalf("expected 3 bodies, got %d", len(system.Bodies))
	}

	names := []string{"Star A", "Star B", "Planet C"}
	for i, name := range names {
		if system.Bodies[i].EnglishName != name {
			t.Errorf("body %d = %q, want %q", i, system.Bodies[i].EnglishName, name)
		}
	}

	moon := system.Bodies[1].Moons
	if len(moon) != 1 || moon[0].EnglishName != "Moon of B" {
		t.Errorf("nested moons wrong: %+v", moon)
	}

	if !system.Bodies[2].IsPlanet || system.Bodies[2].MeanRadius != 6371 {
		t.Errorf("planet fields wrong: %+v", system.Bodies[2])
	}
}

func TestYAMLParseSystemMetadata(t *testing.T) {
	yf := NewYAMLFormat()

	metadata, err := yf.ParseSystemMetadata([]byte(validSingleStarYAML))
	if err != nil {
		t.Fatalf("ParseSystemMetadata: %v", err)
	}

	if metadata.SystemName != "Solo System" || metadata.Description != "A single-star system" ||
		metadata.DiscoveryYear != "2020" || metadata.Distance != "12 light-years" ||
		metadata.Galaxy != "Milky Way" {
		t.Errorf("metadata parsed incorrectly: %+v", metadata)
	}

	// A metadata-only document without bodies still satisfies metadata loading,
	// mirroring the JSON handler which only requires systemName for metadata.
	metadataOnly := "systemName: Metadata Only\ndescription: no bodies here\n"
	metadata, err = yf.ParseSystemMetadata([]byte(metadataOnly))
	if err != nil {
		t.Fatalf("metadata-only document failed: %v", err)
	}
	if metadata.SystemName != "Metadata Only" {
		t.Errorf("SystemName = %q", metadata.SystemName)
	}
}

func TestYAMLSupportsFlowStyleAndUTF8(t *testing.T) {
	yf := NewYAMLFormat()

	flow := `
systemName: 流れ星システム
description: 日本語と絵文字を含む説明 ☄
galaxy: 天の川
bodies:
  - {englishName: ほし, bodyType: Star, mass: {massValue: 2.5, massExponent: 29}}
`

	system, err := yf.ParseSystemData([]byte(flow))
	if err != nil {
		t.Fatalf("ParseSystemData: %v", err)
	}

	if system.SystemName != "流れ星システム" || system.Galaxy != "天の川" {
		t.Errorf("UTF-8 metadata wrong: %+v", system)
	}
	if !strings.Contains(system.Description, "☄") {
		t.Errorf("emoji lost from description: %q", system.Description)
	}
	if system.Bodies[0].Mass.MassValue != 2.5 || system.Bodies[0].Mass.MassExponent != 29 {
		t.Errorf("flow-style nested mass wrong: %+v", system.Bodies[0].Mass)
	}
}

func TestYAMLRejectsEmptyDocuments(t *testing.T) {
	yf := NewYAMLFormat()

	emptyDocs := map[string]string{
		"empty string":    "",
		"whitespace":      "   \n\t  \n",
		"comment only":    "# just a comment\n",
		"explicit marker": "---\n",
		"null scalar":     "null\n",
		"tilde null":      "~\n",
	}

	for name, src := range emptyDocs {
		if err := yf.ValidateFormat([]byte(src)); err == nil {
			t.Errorf("%s: expected empty document to be rejected", name)
		} else if _, err := yf.ParseSystemData([]byte(src)); err == nil {
			t.Errorf("%s: expected ParseSystemData to reject empty document", name)
		}
	}
}

func TestYAMLRejectsMultiDocument(t *testing.T) {
	yf := NewYAMLFormat()

	multi := "systemName: First\nbodies:\n  - englishName: a\n---\nsystemName: Second\n"
	err := yf.ValidateFormat([]byte(multi))
	if err == nil {
		t.Fatal("expected multi-document YAML to be rejected")
	}

	// Position points at the "---" marker that starts the second document.
	requireContains(t, err.Error(), "line 4", "column 1", "multiple YAML documents")
}

func TestYAMLRejectsDuplicateKeys(t *testing.T) {
	yf := NewYAMLFormat()

	duplicateTop := "systemName: First\nsystemName: Second\nbodies: []\n"
	err := yf.ValidateFormat([]byte(duplicateTop))
	if err == nil {
		t.Fatal("expected duplicate top-level key to be rejected")
	}
	requireContains(t, err.Error(), "line 2", "column 1", "duplicate mapping key", "systemName")

	duplicateNested := "systemName: Sys\nbodies:\n  - englishName: b\n    englishName: c\n"
	err = yf.ValidateFormat([]byte(duplicateNested))
	if err == nil {
		t.Fatal("expected duplicate nested key to be rejected")
	}
	requireContains(t, err.Error(), "line 4", "column 5", "duplicate mapping key", "englishName")

	duplicateDeep := "systemName: Sys\nbodies:\n  - englishName: b\n    mass:\n      massValue: 1.0\n      massValue: 2.0\n"
	_, err = yf.ParseSystemData([]byte(duplicateDeep))
	if err == nil {
		t.Fatal("expected duplicate key inside mass mapping to be rejected")
	}
	requireContains(t, err.Error(), "duplicate mapping key", "massValue")
}

func TestYAMLRejectsUnknownTopLevelField(t *testing.T) {
	yf := NewYAMLFormat()

	unknown := "systemName: Sys\nbodies: []\nbogusField: true\n"
	err := yf.ValidateFormat([]byte(unknown))
	if err == nil {
		t.Fatal("expected unknown top-level field to be rejected")
	}
	requireContains(t, err.Error(), "line 3", "column 1", "unknown top-level field", "bogusField")

	// Unknown nested fields stay tolerated so YAML matches JSON's lenient decoding.
	nestedExtra := "systemName: Sys\nbodies:\n  - englishName: b\n    equilibriumTemperature: 234\n"
	if err := yf.ValidateFormat([]byte(nestedExtra)); err != nil {
		t.Errorf("unknown nested fields should be ignored like JSON, got %v", err)
	}
}

func TestYAMLRejectsWrongScalarTypes(t *testing.T) {
	yf := NewYAMLFormat()

	// Nested numeric field given a string: yaml.v3 type error, enriched with column.
	badNumeric := "systemName: Sys\nbodies:\n  - englishName: b\n    meanRadius: \"abc\"\n"
	_, err := yf.ParseSystemData([]byte(badNumeric))
	if err == nil {
		t.Fatal("expected wrong scalar type to be rejected")
	}
	requireContains(t, err.Error(), "line 4", "column", "cannot unmarshal")

	// bodies must be a sequence.
	bodiesScalar := "systemName: Sys\nbodies: nope\n"
	if err := yf.ValidateFormat([]byte(bodiesScalar)); err == nil {
		t.Fatal("expected scalar bodies to be rejected")
	} else {
		requireContains(t, err.Error(), "line 2", "column 9", "bodies")
	}

	// Metadata fields must be scalars.
	nameMapping := "systemName:\n  nested: 1\nbodies: []\n"
	if err := yf.ValidateFormat([]byte(nameMapping)); err == nil {
		t.Fatal("expected mapping systemName to be rejected")
	} else {
		requireContains(t, err.Error(), "line 2", "column 3", "systemName")
	}

	// A sequence entry that is not a mapping.
	nonMappingBody := "systemName: Sys\nbodies:\n  - just-a-string\n"
	if err := yf.ValidateFormat([]byte(nonMappingBody)); err == nil {
		t.Fatal("expected non-mapping body entry to be rejected")
	} else {
		requireContains(t, err.Error(), "line 3", "mapping")
	}

	// A string scalar cannot be loaded into a typed numeric/bool field.
	wrongBool := "systemName: Sys\nbodies:\n  - englishName: b\n    isPlanet: \"maybe\"\n"
	if _, err := yf.ParseSystemData([]byte(wrongBool)); err == nil {
		t.Fatal("expected string scalar in bool field to be rejected")
	} else {
		requireContains(t, err.Error(), "line 4", "column", "cannot unmarshal")
	}
}

func TestYAMLRejectsSyntaxErrorsAndNonMappingRoot(t *testing.T) {
	yf := NewYAMLFormat()

	// Tab indentation is invalid YAML even though the line contains a colon.
	tabIndented := "foo: bar\n\tbadtab: 1\n"
	if err := yf.ValidateFormat([]byte(tabIndented)); err == nil {
		t.Fatal("expected tab-indented content to be rejected")
	} else if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("syntax error should carry a line position: %v", err)
	}

	scalarRoot := "just a scalar\n"
	if err := yf.ValidateFormat([]byte(scalarRoot)); err == nil {
		t.Fatal("expected scalar root to be rejected")
	} else {
		requireContains(t, err.Error(), "line 1", "column 1", "root must be a mapping")
	}

	sequenceRoot := "- a\n- b\n"
	if err := yf.ValidateFormat([]byte(sequenceRoot)); err == nil {
		t.Fatal("expected sequence root to be rejected")
	}
}

func TestYAMLValidateFormatRequiresFields(t *testing.T) {
	yf := NewYAMLFormat()

	missingBodies := "systemName: Sys\n"
	if err := yf.ValidateFormat([]byte(missingBodies)); err == nil ||
		!strings.Contains(err.Error(), "missing required field: bodies") {
		t.Errorf("missing bodies should be reported, got %v", err)
	}

	missingName := "bodies:\n  - englishName: b\n"
	if err := yf.ValidateFormat([]byte(missingName)); err == nil ||
		!strings.Contains(err.Error(), "missing required field: systemName") {
		t.Errorf("missing systemName should be reported, got %v", err)
	}

	// Full parse applies the same shared validation as JSON.
	if _, err := yf.ParseSystemData([]byte(missingBodies)); err == nil ||
		!strings.Contains(err.Error(), "at least one celestial body") {
		t.Errorf("shared body validation not applied to YAML, got %v", err)
	}

	emptyName := "systemName: \"   \"\nbodies:\n  - englishName: b\n"
	if _, err := yf.ParseSystemData([]byte(emptyName)); err == nil ||
		!strings.Contains(err.Error(), "systemName cannot be empty") {
		t.Errorf("shared systemName validation not applied to YAML, got %v", err)
	}

	bodyWithoutName := "systemName: Sys\nbodies:\n  - meanRadius: 100\n"
	if _, err := yf.ParseSystemData([]byte(bodyWithoutName)); err == nil ||
		!strings.Contains(err.Error(), "missing englishName") {
		t.Errorf("shared body validation not applied to YAML, got %v", err)
	}
}

func TestDetectFormatParsesInsteadOfGuessing(t *testing.T) {
	registry := NewFormatRegistry()

	yamlHandler, err := registry.DetectFormat([]byte(validMultiStarYAML))
	if err != nil {
		t.Fatalf("YAML content should be detected: %v", err)
	}
	if yamlHandler.GetFormatName() != "YAML" {
		t.Errorf("detected %s, want YAML", yamlHandler.GetFormatName())
	}

	jsonHandler, err := registry.DetectFormat([]byte(`{"systemName":"J","bodies":[]}`))
	if err != nil {
		t.Fatalf("JSON content should be detected: %v", err)
	}
	if jsonHandler.GetFormatName() != "JSON" {
		t.Errorf("detected %s, want JSON", jsonHandler.GetFormatName())
	}

	// Colon-bearing content that is not parseable YAML must not be accepted.
	invalid := "a: b: c\n"
	if _, err := registry.DetectFormat([]byte(invalid)); err == nil {
		t.Fatal("invalid colon-bearing content must fail real parsing")
	}
}

func TestJSONAndYAMLHaveEquivalentBehavior(t *testing.T) {
	jsonDoc := `{
		"systemName": "Parity System",
		"description": "same data, two formats",
		"discoveryYear": "2021",
		"distance": "8.6 ly",
		"galaxy": "Milky Way",
		"bodies": [
			{
				"englishName": "Parity Star",
				"bodyType": "Star",
				"isPlanet": false,
				"meanRadius": 500000,
				"mass": {"massValue": 0.8, "massExponent": 30},
				"moons": [{"englishName": "Parity Moon"}]
			}
		]
	}`

	yamlDoc := `
systemName: Parity System
description: same data, two formats
discoveryYear: "2021"
distance: 8.6 ly
galaxy: Milky Way
bodies:
  - englishName: Parity Star
    bodyType: Star
    isPlanet: false
    meanRadius: 500000
    mass:
      massValue: 0.8
      massExponent: 30
    moons:
      - englishName: Parity Moon
`

	jf := NewJSONFormat()
	yf := NewYAMLFormat()

	jsonSystem, err := jf.ParseSystemData([]byte(jsonDoc))
	if err != nil {
		t.Fatalf("JSON ParseSystemData: %v", err)
	}

	yamlSystem, err := yf.ParseSystemData([]byte(yamlDoc))
	if err != nil {
		t.Fatalf("YAML ParseSystemData: %v", err)
	}

	if !reflect.DeepEqual(*jsonSystem, *yamlSystem) {
		t.Errorf("full data diverged:\nJSON: %+v\nYAML: %+v", *jsonSystem, *yamlSystem)
	}

	jsonMeta, err := jf.ParseSystemMetadata([]byte(jsonDoc))
	if err != nil {
		t.Fatalf("JSON ParseSystemMetadata: %v", err)
	}

	yamlMeta, err := yf.ParseSystemMetadata([]byte(yamlDoc))
	if err != nil {
		t.Fatalf("YAML ParseSystemMetadata: %v", err)
	}

	if *jsonMeta != *yamlMeta {
		t.Errorf("metadata diverged:\nJSON: %+v\nYAML: %+v", *jsonMeta, *yamlMeta)
	}

	// JSON payloads are valid YAML too, and the YAML handler parses them identically.
	yamlFromJSON, err := yf.ParseSystemData([]byte(jsonDoc))
	if err != nil {
		t.Fatalf("YAML handler should accept JSON syntax: %v", err)
	}
	if !reflect.DeepEqual(*yamlFromJSON, *jsonSystem) {
		t.Errorf("YAML-handled JSON payload diverged: %+v", yamlFromJSON)
	}
}

func TestSystemFileSizeLimitAppliesToBothFormats(t *testing.T) {
	valid := "systemName: Big\nbodies:\n  - englishName: b\n"

	yf := NewYAMLFormat()
	jf := NewJSONFormat()

	oversized := make([]byte, MaxSystemFileSize+1)
	for _, handler := range []FileFormat{yf, jf} {
		if err := handler.ValidateFormat(oversized); err == nil ||
			!strings.Contains(err.Error(), "too large") {
			t.Errorf("%s should reject oversized content, got %v", handler.GetFormatName(), err)
		}
		if _, err := handler.ParseSystemData(oversized); err == nil {
			t.Errorf("%s ParseSystemData should reject oversized content", handler.GetFormatName())
		}
		if _, err := handler.ParseSystemMetadata(oversized); err == nil {
			t.Errorf("%s ParseSystemMetadata should reject oversized content", handler.GetFormatName())
		}
	}

	// A document exactly at the limit (padded with comments) must still parse.
	padding := MaxSystemFileSize - len(valid) - 3
	atLimit := "# " + strings.Repeat("x", padding) + "\n" + valid
	if len(atLimit) != MaxSystemFileSize {
		t.Fatalf("test setup: got %d bytes, want %d", len(atLimit), MaxSystemFileSize)
	}
	if err := yf.ValidateFormat([]byte(atLimit)); err != nil {
		t.Errorf("document at the size limit should be accepted: %v", err)
	}
}
