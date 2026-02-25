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

// createSeededQuoteDB creates a migrated, seeded database with one vendor, one
// project, and one quote. Returns the DB path, quote ID, project ID, and
// vendor ID.
func createSeededQuoteDB(t *testing.T) (dbPath string, quoteID, projectID, vendorID uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "quotes.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	vendor := data.Vendor{Name: "Acme Plumbing"}
	require.NoError(t, store.CreateVendor(&vendor))

	project := data.Project{
		Title:         "Kitchen Remodel",
		ProjectTypeID: 1,
		Status:        data.ProjectStatusPlanned,
	}
	require.NoError(t, store.CreateProject(&project))

	laborCents := int64(50000)
	materialsCents := int64(30000)
	quote := data.Quote{
		ProjectID:      project.ID,
		TotalCents:     150000,
		LaborCents:     &laborCents,
		MaterialsCents: &materialsCents,
		Notes:          "Initial estimate",
	}
	require.NoError(t, store.CreateQuote(&quote, vendor))
	require.NoError(t, store.Close())
	return path, quote.ID, project.ID, vendor.ID
}

func TestQuotesList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _, projectID, vendorID := createSeededQuoteDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Kitchen Remodel")
		assert.Contains(t, got, "Acme Plumbing")
		assert.Contains(t, got, "$1,500.00")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list --json failed: %s", out)

		var items []quoteJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Kitchen Remodel", items[0].Project)
		assert.Equal(t, "Acme Plumbing", items[0].Vendor)
		assert.Equal(t, int64(150000), items[0].TotalCents)
	})

	t.Run("FilterByProject", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "list", "--project-id", idString(projectID), "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list --project-id failed: %s", out)
		assert.Contains(t, string(out), "Kitchen Remodel")
	})

	t.Run("FilterByVendor", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "list", "--vendor-id", idString(vendorID), "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list --vendor-id failed: %s", out)
		assert.Contains(t, string(out), "Acme Plumbing")
	})

	t.Run("FilterByProjectNoResults", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "list", "--project-id", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list --project-id 9999 failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes (default list) failed: %s", out)
		assert.Contains(t, string(out), "Kitchen Remodel")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := filepath.Join(t.TempDir(), "empty.db")
		store, err := data.Open(emptyDB)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "quotes", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestQuotesAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath, _, projectID, vendorID := createSeededQuoteDB(t)

		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--project-id", idString(projectID),
			"--vendor-id", idString(vendorID),
			"--total", "2500.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected quote ID in output")

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		items, err := store.ListQuotes(false)
		require.NoError(t, err)
		require.Len(t, items, 2)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath, _, projectID, vendorID := createSeededQuoteDB(t)

		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--project-id", idString(projectID),
			"--vendor-id", idString(vendorID),
			"--total", "5000.00",
			"--labor", "3000.00",
			"--materials", "1500.00",
			"--other", "500.00",
			"--received-date", "2026-02-20",
			"--notes", "Detailed breakdown",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes add (all flags) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		items, err := store.ListQuotes(false)
		require.NoError(t, err)
		// Find the new quote (most recently updated should be first).
		var found *data.Quote
		for i := range items {
			if items[i].Notes == "Detailed breakdown" {
				found = &items[i]
				break
			}
		}
		require.NotNil(t, found, "new quote not found")
		assert.Equal(t, int64(500000), found.TotalCents)
		require.NotNil(t, found.LaborCents)
		assert.Equal(t, int64(300000), *found.LaborCents)
		require.NotNil(t, found.MaterialsCents)
		assert.Equal(t, int64(150000), *found.MaterialsCents)
		require.NotNil(t, found.OtherCents)
		assert.Equal(t, int64(50000), *found.OtherCents)
		require.NotNil(t, found.ReceivedDate)
		assert.Equal(t, "2026-02-20", found.ReceivedDate.Format("2006-01-02"))
	})

	t.Run("MissingRequired", func(t *testing.T) {
		dbPath, _, _, vendorID := createSeededQuoteDB(t)
		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--vendor-id", idString(vendorID),
			"--total", "100.00",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidTotal", func(t *testing.T) {
		dbPath, _, projectID, vendorID := createSeededQuoteDB(t)
		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--project-id", idString(projectID),
			"--vendor-id", idString(vendorID),
			"--total", "not-a-number",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "parse total")
	})

	t.Run("InvalidDate", func(t *testing.T) {
		dbPath, _, projectID, vendorID := createSeededQuoteDB(t)
		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--project-id", idString(projectID),
			"--vendor-id", idString(vendorID),
			"--total", "100.00",
			"--received-date", "Feb 20 2026",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "received-date")
	})

	t.Run("BadVendorID", func(t *testing.T) {
		dbPath, _, projectID, _ := createSeededQuoteDB(t)
		cmd := exec.Command(bin, "quotes", "add", //nolint:gosec // test binary
			"--project-id", idString(projectID),
			"--vendor-id", "9999",
			"--total", "100.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "vendor")
	})
}

func TestQuotesShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, quoteID, _, _ := createSeededQuoteDB(t)
	qID := idString(quoteID)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "show", qID, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Kitchen Remodel")
		assert.Contains(t, got, "Acme Plumbing")
		assert.Contains(t, got, "$1,500.00")
		assert.Contains(t, got, "$500.00")
		assert.Contains(t, got, "$300.00")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "show", qID, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes show --json failed: %s", out)

		var item quoteJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Kitchen Remodel", item.Project)
		assert.Equal(t, "Acme Plumbing", item.Vendor)
		assert.Equal(t, int64(150000), item.TotalCents)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "quotes", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestQuotesUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateTotal", func(t *testing.T) {
		dbPath, quoteID, _, _ := createSeededQuoteDB(t)
		qID := idString(quoteID)

		cmd := exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--total", "2000.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		quote, err := store.GetQuote(quoteID)
		require.NoError(t, err)
		assert.Equal(t, int64(200000), quote.TotalCents)
	})

	t.Run("UpdateNotes", func(t *testing.T) {
		dbPath, quoteID, _, _ := createSeededQuoteDB(t)
		qID := idString(quoteID)

		cmd := exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--notes", "Revised estimate",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes update notes failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		quote, err := store.GetQuote(quoteID)
		require.NoError(t, err)
		assert.Equal(t, "Revised estimate", quote.Notes)
	})

	t.Run("UpdateReceivedDate", func(t *testing.T) {
		dbPath, quoteID, _, _ := createSeededQuoteDB(t)
		qID := idString(quoteID)

		cmd := exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--received-date", "2026-03-01",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "quotes update received-date failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		quote, err := store.GetQuote(quoteID)
		require.NoError(t, err)
		require.NotNil(t, quote.ReceivedDate)
		assert.Equal(t, "2026-03-01", quote.ReceivedDate.Format("2006-01-02"))
	})

	t.Run("ClearReceivedDate", func(t *testing.T) {
		dbPath, quoteID, _, _ := createSeededQuoteDB(t)
		qID := idString(quoteID)

		// First set a received date.
		cmd := exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--received-date", "2026-03-01",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "set received-date failed: %s", out)

		// Then clear it.
		cmd = exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--received-date", "",
			"--db-path", dbPath,
		)
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, "clear received-date failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		quote, err := store.GetQuote(quoteID)
		require.NoError(t, err)
		assert.Nil(t, quote.ReceivedDate)
	})

	t.Run("InvalidTotal", func(t *testing.T) {
		dbPath, quoteID, _, _ := createSeededQuoteDB(t)
		qID := idString(quoteID)

		cmd := exec.Command(bin, "quotes", "update", qID, //nolint:gosec // test binary
			"--total", "bad",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "parse total")
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _, _, _ := createSeededQuoteDB(t)
		cmd := exec.Command(bin, "quotes", "update", "9999", //nolint:gosec // test binary
			"--total", "100.00",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestQuotesDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, quoteID, _, _ := createSeededQuoteDB(t)
	qID := idString(quoteID)

	cmd := exec.Command(bin, "quotes", "delete", qID, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "quotes delete failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	items, err := store.ListQuotes(false)
	require.NoError(t, err)
	assert.Empty(t, items)

	all, err := store.ListQuotes(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
