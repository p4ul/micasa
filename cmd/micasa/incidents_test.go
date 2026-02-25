// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpcloud/micasa/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSeededDB creates a migrated, seeded database with one incident and
// returns the DB path and the incident ID.
func createSeededDB(t *testing.T) (string, uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "incidents.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	item := data.Incident{
		Title:    "Leaky faucet",
		Status:   data.IncidentStatusOpen,
		Severity: data.IncidentSeverityUrgent,
		Location: "Kitchen",
	}
	require.NoError(t, store.CreateIncident(&item))
	require.NoError(t, store.Close())
	return path, item.ID
}

func TestIncidentsList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _ := createSeededDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Leaky faucet")
		assert.Contains(t, got, "urgent")
		assert.Contains(t, got, "Kitchen")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents list --json failed: %s", out)

		var items []incidentJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Leaky faucet", items[0].Title)
		assert.Equal(t, "urgent", items[0].Severity)
	})

	t.Run("FilterStatus", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "list", "--status", "resolved", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents list --status failed: %s", out)
		assert.NotContains(t, string(out), "Leaky faucet")
	})

	t.Run("FilterStatusInvalid", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "list", "--status", "bogus", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid status")
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents (default list) failed: %s", out)
		assert.Contains(t, string(out), "Leaky faucet")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := filepath.Join(t.TempDir(), "empty.db")
		store, err := data.Open(emptyDB)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "incidents", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestIncidentsAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "incidents", "add", //nolint:gosec // test binary
			"--title", "Broken window",
			"--severity", "soon",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected incident ID in output")

		// Verify it was persisted.
		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListIncidents(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Broken window", items[0].Title)
		assert.Equal(t, data.IncidentStatusOpen, items[0].Status)
		assert.Equal(t, data.IncidentSeveritySoon, items[0].Severity)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add-full.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "incidents", "add", //nolint:gosec // test binary
			"--title", "Roof leak",
			"--severity", "urgent",
			"--description", "Water dripping from ceiling",
			"--location", "Attic",
			"--notes", "Check after rain",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents add (all flags) failed: %s", out)

		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListIncidents(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Roof leak", items[0].Title)
		assert.Equal(t, "Water dripping from ceiling", items[0].Description)
		assert.Equal(t, "Attic", items[0].Location)
		assert.Equal(t, "Check after rain", items[0].Notes)
	})

	t.Run("MissingTitle", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "miss.db")
		cmd := exec.Command(bin, "incidents", "add", //nolint:gosec // test binary
			"--severity", "soon",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidSeverity", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "bad-sev.db")
		cmd := exec.Command(bin, "incidents", "add", //nolint:gosec // test binary
			"--title", "Test",
			"--severity", "critical",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "severity")
	})
}

func TestIncidentsShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededDB(t)
	idStr := idString(id)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "show", idStr, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Leaky faucet")
		assert.Contains(t, got, "urgent")
		assert.Contains(t, got, "Kitchen")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "show", idStr, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents show --json failed: %s", out)

		var item incidentJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Leaky faucet", item.Title)
		assert.Equal(t, "Kitchen", item.Location)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "incidents", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestIncidentsUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateTitle", func(t *testing.T) {
		dbPath, id := createSeededDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "incidents", "update", idStr, //nolint:gosec // test binary
			"--title", "Fixed faucet",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetIncident(id)
		require.NoError(t, err)
		assert.Equal(t, "Fixed faucet", item.Title)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		dbPath, id := createSeededDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "incidents", "update", idStr, //nolint:gosec // test binary
			"--status", "in_progress",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "incidents update status failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetIncident(id)
		require.NoError(t, err)
		assert.Equal(t, data.IncidentStatusInProgress, item.Status)
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		dbPath, id := createSeededDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "incidents", "update", idStr, //nolint:gosec // test binary
			"--status", "bogus",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid status")
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _ := createSeededDB(t)
		cmd := exec.Command(bin, "incidents", "update", "9999", //nolint:gosec // test binary
			"--title", "Nope",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestIncidentsDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededDB(t)
	idStr := idString(id)

	cmd := exec.Command(bin, "incidents", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "incidents delete failed: %s", out)

	// Verify soft-deleted: not in normal list.
	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	items, err := store.ListIncidents(false)
	require.NoError(t, err)
	assert.Empty(t, items)

	// Still visible with includeDeleted.
	all, err := store.ListIncidents(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestIncidentsResolve(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededDB(t)
	idStr := idString(id)

	cmd := exec.Command(bin, "incidents", "resolve", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "incidents resolve failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	item, err := store.GetIncident(id)
	require.NoError(t, err)
	assert.Equal(t, data.IncidentStatusResolved, item.Status)
	assert.NotNil(t, item.DateResolved)
}

func idString(id uint) string {
	return fmt.Sprintf("%d", id)
}
