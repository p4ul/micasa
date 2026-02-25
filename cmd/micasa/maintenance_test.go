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

func createSeededMaintenanceDB(t *testing.T) (string, uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "maintenance.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	// Look up HVAC category.
	cats, err := store.MaintenanceCategories()
	require.NoError(t, err)
	var hvacID uint
	for _, c := range cats {
		if c.Name == "HVAC" {
			hvacID = c.ID
			break
		}
	}
	require.NotZero(t, hvacID, "HVAC category must exist")

	cost := int64(15000)
	item := data.MaintenanceItem{
		Name:           "Replace air filter",
		CategoryID:     hvacID,
		IntervalMonths: 3,
		CostCents:      &cost,
		Notes:          "Use MERV-13",
	}
	require.NoError(t, store.CreateMaintenance(&item))
	require.NoError(t, store.Close())
	return path, item.ID
}

func createEmptyMaintenanceDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.Close())
	return path
}

func TestMaintenanceList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _ := createSeededMaintenanceDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Replace air filter")
		assert.Contains(t, got, "HVAC")
		assert.Contains(t, got, "3mo")
		assert.Contains(t, got, "$150.00")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance list --json failed: %s", out)

		var items []maintenanceJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Replace air filter", items[0].Name)
		assert.Equal(t, "HVAC", items[0].Category)
		assert.Equal(t, 3, items[0].IntervalMonths)
		require.NotNil(t, items[0].CostCents)
		assert.Equal(t, int64(15000), *items[0].CostCents)
	})

	t.Run("FilterCategory", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "list", "--category", "Plumbing", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance list --category failed: %s", out)
		assert.NotContains(t, string(out), "Replace air filter")
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance (default list) failed: %s", out)
		assert.Contains(t, string(out), "Replace air filter")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestMaintenanceAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--name", "Clean gutters",
			"--category", "Exterior",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected maintenance item ID in output")

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		items, err := store.ListMaintenance(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Clean gutters", items[0].Name)
		assert.Equal(t, "Exterior", items[0].Category.Name)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--name", "Replace furnace filter",
			"--category", "HVAC",
			"--interval-months", "6",
			"--cost", "25.00",
			"--notes", "Buy in bulk",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance add (all flags) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		items, err := store.ListMaintenance(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Replace furnace filter", items[0].Name)
		assert.Equal(t, 6, items[0].IntervalMonths)
		require.NotNil(t, items[0].CostCents)
		assert.Equal(t, int64(2500), *items[0].CostCents)
		assert.Equal(t, "Buy in bulk", items[0].Notes)
	})

	t.Run("MissingName", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--category", "HVAC",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("MissingCategory", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--name", "Test",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidCategory", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--name", "Test",
			"--category", "Nonexistent",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "unknown category")
	})

	t.Run("InvalidCost", func(t *testing.T) {
		dbPath := createEmptyMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "add", //nolint:gosec // test binary
			"--name", "Test",
			"--category", "HVAC",
			"--cost", "notanumber",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid cost")
	})
}

func TestMaintenanceShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededMaintenanceDB(t)
	idStr := fmt.Sprintf("%d", id)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "show", idStr, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Replace air filter")
		assert.Contains(t, got, "HVAC")
		assert.Contains(t, got, "3 months")
		assert.Contains(t, got, "$150.00")
		assert.Contains(t, got, "Use MERV-13")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "show", idStr, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance show --json failed: %s", out)

		var item maintenanceJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Replace air filter", item.Name)
		assert.Equal(t, "HVAC", item.Category)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "maintenance", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestMaintenanceUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateName", func(t *testing.T) {
		dbPath, id := createSeededMaintenanceDB(t)
		idStr := fmt.Sprintf("%d", id)

		cmd := exec.Command(bin, "maintenance", "update", idStr, //nolint:gosec // test binary
			"--name", "Replace HEPA filter",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetMaintenance(id)
		require.NoError(t, err)
		assert.Equal(t, "Replace HEPA filter", item.Name)
	})

	t.Run("UpdateCategory", func(t *testing.T) {
		dbPath, id := createSeededMaintenanceDB(t)
		idStr := fmt.Sprintf("%d", id)

		cmd := exec.Command(bin, "maintenance", "update", idStr, //nolint:gosec // test binary
			"--category", "Plumbing",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance update category failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetMaintenance(id)
		require.NoError(t, err)
		assert.Equal(t, "Plumbing", item.Category.Name)
	})

	t.Run("UpdateCost", func(t *testing.T) {
		dbPath, id := createSeededMaintenanceDB(t)
		idStr := fmt.Sprintf("%d", id)

		cmd := exec.Command(bin, "maintenance", "update", idStr, //nolint:gosec // test binary
			"--cost", "$200.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "maintenance update cost failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetMaintenance(id)
		require.NoError(t, err)
		require.NotNil(t, item.CostCents)
		assert.Equal(t, int64(20000), *item.CostCents)
	})

	t.Run("InvalidCategory", func(t *testing.T) {
		dbPath, id := createSeededMaintenanceDB(t)
		idStr := fmt.Sprintf("%d", id)

		cmd := exec.Command(bin, "maintenance", "update", idStr, //nolint:gosec // test binary
			"--category", "Nonexistent",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "unknown category")
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _ := createSeededMaintenanceDB(t)
		cmd := exec.Command(bin, "maintenance", "update", "9999", //nolint:gosec // test binary
			"--name", "Nope",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestMaintenanceDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededMaintenanceDB(t)
	idStr := fmt.Sprintf("%d", id)

	cmd := exec.Command(bin, "maintenance", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "maintenance delete failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	items, err := store.ListMaintenance(false)
	require.NoError(t, err)
	assert.Empty(t, items)

	all, err := store.ListMaintenance(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
