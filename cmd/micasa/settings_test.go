// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpcloud/micasa/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSettingsDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.Close())
	return path
}

func TestSettingsListDefault(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "list", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "settings list failed: %s", out)
	got := string(out)
	assert.Contains(t, got, "units.system")
	assert.Contains(t, got, data.DefaultUnitSystem)
	assert.Contains(t, got, "units.currency")
	assert.Contains(t, got, data.DefaultCurrency)
}

func TestSettingsListDefaultSubcommand(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "settings (default list) failed: %s", out)
	assert.Contains(t, string(out), "units.system")
}

func TestSettingsSetUnits(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	// Set to imperial.
	cmd := exec.Command(bin, "settings", "set", "units", "imperial", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "settings set units failed: %s", out)
	assert.Empty(t, strings.TrimSpace(string(out)), "silence is success")

	// Verify via list.
	cmd = exec.Command(bin, "settings", "list", "--db-path", dbPath) //nolint:gosec // test binary
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "settings list failed: %s", out)
	assert.Contains(t, string(out), "imperial")
}

func TestSettingsSetCurrency(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	// Set to USD.
	cmd := exec.Command(bin, "settings", "set", "currency", "USD", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "settings set currency failed: %s", out)
	assert.Empty(t, strings.TrimSpace(string(out)))

	// Verify persistence by reopening store.
	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	val, err := store.GetCurrency()
	require.NoError(t, err)
	assert.Equal(t, "USD", val)
}

func TestSettingsSetCurrencyLowercase(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "set", "currency", "eur", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "settings set currency lowercase failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	val, err := store.GetCurrency()
	require.NoError(t, err)
	assert.Equal(t, "EUR", val, "should normalize to uppercase")
}

func TestSettingsSetUnitsInvalid(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "set", "units", "cubits", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "invalid unit system")
}

func TestSettingsSetCurrencyInvalid(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "set", "currency", "DOGECOIN", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "invalid currency")
}

func TestSettingsSetUnknownKey(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath := createSettingsDB(t)

	cmd := exec.Command(bin, "settings", "set", "theme", "dark", "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "unknown setting")
}
