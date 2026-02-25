// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpcloud/micasa/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSeededProjectDB(t *testing.T) (string, uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	types, err := store.ProjectTypes()
	require.NoError(t, err)
	require.NotEmpty(t, types)

	budget := int64(500000)
	item := data.Project{
		Title:         "Kitchen remodel",
		ProjectTypeID: types[0].ID,
		Status:        data.ProjectStatusPlanned,
		Description:   "Full kitchen renovation",
		BudgetCents:   &budget,
	}
	require.NoError(t, store.CreateProject(&item))
	require.NoError(t, store.Close())
	return path, item.ID
}

func TestProjectsList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _ := createSeededProjectDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Kitchen remodel")
		assert.Contains(t, got, "planned")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects list --json failed: %s", out)

		var items []projectJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Kitchen remodel", items[0].Title)
		assert.Equal(t, "planned", items[0].Status)
	})

	t.Run("FilterStatus", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "list", "--status", "completed", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects list --status failed: %s", out)
		assert.NotContains(t, string(out), "Kitchen remodel")
	})

	t.Run("FilterStatusInvalid", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "list", "--status", "bogus", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid status")
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects (default list) failed: %s", out)
		assert.Contains(t, string(out), "Kitchen remodel")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := filepath.Join(t.TempDir(), "empty.db")
		store, err := data.Open(emptyDB)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "projects", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestProjectsAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--title", "New deck",
			"--type", "Exterior",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected project ID in output")

		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListProjects(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "New deck", items[0].Title)
		assert.Equal(t, data.ProjectStatusIdeating, items[0].Status)
		assert.Equal(t, "Exterior", items[0].ProjectType.Name)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add-full.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--title", "Bathroom redo",
			"--type", "Remodel",
			"--status", "planned",
			"--description", "Master bath renovation",
			"--budget", "15000.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects add (all flags) failed: %s", out)

		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListProjects(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Bathroom redo", items[0].Title)
		assert.Equal(t, data.ProjectStatusPlanned, items[0].Status)
		assert.Equal(t, "Remodel", items[0].ProjectType.Name)
		assert.Equal(t, "Master bath renovation", items[0].Description)
		require.NotNil(t, items[0].BudgetCents)
		assert.Equal(t, int64(1500000), *items[0].BudgetCents)
	})

	t.Run("MissingTitle", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "miss.db")
		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--type", "Flooring",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("MissingType", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "miss-type.db")
		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--title", "Test",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidType", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "bad-type.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--title", "Test",
			"--type", "Imaginary",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "unknown project type")
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "bad-status.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "projects", "add", //nolint:gosec // test binary
			"--title", "Test",
			"--type", "Flooring",
			"--status", "nonsense",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid status")
	})
}

func TestProjectsShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededProjectDB(t)
	idStr := idString(id)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "show", idStr, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Kitchen remodel")
		assert.Contains(t, got, "planned")
		assert.Contains(t, got, "$5,000.00")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "show", idStr, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects show --json failed: %s", out)

		var item projectJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Kitchen remodel", item.Title)
		assert.Equal(t, "planned", item.Status)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestProjectsUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateTitle", func(t *testing.T) {
		dbPath, id := createSeededProjectDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "projects", "update", idStr, //nolint:gosec // test binary
			"--title", "Updated kitchen",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetProject(id)
		require.NoError(t, err)
		assert.Equal(t, "Updated kitchen", item.Title)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		dbPath, id := createSeededProjectDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "projects", "update", idStr, //nolint:gosec // test binary
			"--status", "underway",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects update status failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetProject(id)
		require.NoError(t, err)
		assert.Equal(t, data.ProjectStatusInProgress, item.Status)
	})

	t.Run("UpdateType", func(t *testing.T) {
		dbPath, id := createSeededProjectDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "projects", "update", idStr, //nolint:gosec // test binary
			"--type", "Plumbing",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects update type failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetProject(id)
		require.NoError(t, err)
		assert.Equal(t, "Plumbing", item.ProjectType.Name)
	})

	t.Run("UpdateBudget", func(t *testing.T) {
		dbPath, id := createSeededProjectDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "projects", "update", idStr, //nolint:gosec // test binary
			"--budget", "7500.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "projects update budget failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetProject(id)
		require.NoError(t, err)
		require.NotNil(t, item.BudgetCents)
		assert.Equal(t, int64(750000), *item.BudgetCents)
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		dbPath, id := createSeededProjectDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "projects", "update", idStr, //nolint:gosec // test binary
			"--status", "bogus",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid status")
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _ := createSeededProjectDB(t)
		cmd := exec.Command(bin, "projects", "update", "9999", //nolint:gosec // test binary
			"--title", "Nope",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestProjectsDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededProjectDB(t)
	idStr := idString(id)

	cmd := exec.Command(bin, "projects", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "projects delete failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	items, err := store.ListProjects(false)
	require.NoError(t, err)
	assert.Empty(t, items)

	all, err := store.ListProjects(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
