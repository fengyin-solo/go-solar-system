package main

import "testing"

func TestParseCLIArgsDefaultsToOnline(t *testing.T) {
	options, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf("parseCLIArgs(nil): %v", err)
	}
	if options.offline {
		t.Error("default mode should be online")
	}
}

func TestParseCLIArgsOfflineLongFlag(t *testing.T) {
	options, err := parseCLIArgs([]string{"--offline"})
	if err != nil {
		t.Fatalf("parseCLIArgs(--offline): %v", err)
	}
	if !options.offline {
		t.Error("--offline should enable offline mode")
	}
}

func TestParseCLIArgsOfflineShortFlag(t *testing.T) {
	options, err := parseCLIArgs([]string{"-offline"})
	if err != nil {
		t.Fatalf("parseCLIArgs(-offline): %v", err)
	}
	if !options.offline {
		t.Error("-offline should enable offline mode")
	}
}

func TestParseCLIArgsRejectsUnknownFlag(t *testing.T) {
	if _, err := parseCLIArgs([]string{"--bogus"}); err == nil {
		t.Error("unknown flag must be rejected")
	}
}
