// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cpcloud/micasa/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSeededApplianceDB creates a migrated, seeded database with one
// appliance and returns the DB path and the appliance ID.
func createSeededApplianceDB(t *testing.T) (string, uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "appliances.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	purchaseDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	warrantyExpiry := time.Date(2028, 6, 15, 0, 0, 0, 0, time.UTC)
	cost := int64(49999)
	item := data.Appliance{
		Name:           "Dishwasher",
		Brand:          "Bosch",
		ModelNumber:    "SHE3AR75UC",
		SerialNumber:   "FD1234567",
		Location:       "Kitchen",
		PurchaseDate:   &purchaseDate,
		WarrantyExpiry: &warrantyExpiry,
		CostCents:      &cost,
	}
	require.NoError(t, store.CreateAppliance(&item))
	require.NoError(t, store.Close())
	return path, item.ID
}

func TestAppliancesList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _ := createSeededApplianceDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Dishwasher")
		assert.Contains(t, got, "Bosch")
		assert.Contains(t, got, "Kitchen")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances list --json failed: %s", out)

		var items []applianceJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Dishwasher", items[0].Name)
		assert.Equal(t, "Bosch", items[0].Brand)
		assert.Equal(t, "2025-06-15", items[0].PurchaseDate)
		assert.Equal(t, "2028-06-15", items[0].WarrantyExpiry)
		assert.Equal(t, int64(49999), *items[0].CostCents)
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances (default list) failed: %s", out)
		assert.Contains(t, string(out), "Dishwasher")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := filepath.Join(t.TempDir(), "empty.db")
		store, err := data.Open(emptyDB)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "appliances", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestAppliancesAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "appliances", "add", //nolint:gosec // test binary
			"--name", "Toaster",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected appliance ID in output")

		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListAppliances(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Toaster", items[0].Name)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "add-full.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "appliances", "add", //nolint:gosec // test binary
			"--name", "Washing Machine",
			"--brand", "LG",
			"--model", "WM3900HWA",
			"--serial", "SN98765",
			"--location", "Laundry Room",
			"--cost", "899.99",
			"--purchase-date", "2025-01-10",
			"--warranty-expiry", "2027-01-10",
			"--notes", "Front loader",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances add (all flags) failed: %s", out)

		store2, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store2.Close() }()
		items, err := store2.ListAppliances(false)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "Washing Machine", items[0].Name)
		assert.Equal(t, "LG", items[0].Brand)
		assert.Equal(t, "WM3900HWA", items[0].ModelNumber)
		assert.Equal(t, "SN98765", items[0].SerialNumber)
		assert.Equal(t, "Laundry Room", items[0].Location)
		require.NotNil(t, items[0].CostCents)
		assert.Equal(t, int64(89999), *items[0].CostCents)
		require.NotNil(t, items[0].PurchaseDate)
		assert.Equal(t, "2025-01-10", items[0].PurchaseDate.Format(time.DateOnly))
		require.NotNil(t, items[0].WarrantyExpiry)
		assert.Equal(t, "2027-01-10", items[0].WarrantyExpiry.Format(time.DateOnly))
		assert.Equal(t, "Front loader", items[0].Notes)
	})

	t.Run("MissingName", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "miss.db")
		cmd := exec.Command(bin, "appliances", "add", //nolint:gosec // test binary
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidCost", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "bad-cost.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "appliances", "add", //nolint:gosec // test binary
			"--name", "Test",
			"--cost", "not-a-number",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "parse cost")
	})

	t.Run("InvalidPurchaseDate", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "bad-date.db")
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.AutoMigrate())
		require.NoError(t, store.SeedDefaults())
		require.NoError(t, store.Close())

		cmd := exec.Command(bin, "appliances", "add", //nolint:gosec // test binary
			"--name", "Test",
			"--purchase-date", "June 15 2025",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "parse purchase date")
	})
}

func TestAppliancesShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededApplianceDB(t)
	idStr := idString(id)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "show", idStr, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Dishwasher")
		assert.Contains(t, got, "Bosch")
		assert.Contains(t, got, "SHE3AR75UC")
		assert.Contains(t, got, "Kitchen")
		assert.Contains(t, got, "$499.99")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "show", idStr, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances show --json failed: %s", out)

		var item applianceJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Dishwasher", item.Name)
		assert.Equal(t, "Bosch", item.Brand)
		assert.Equal(t, "SHE3AR75UC", item.ModelNumber)
		assert.Equal(t, "Kitchen", item.Location)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "appliances", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestAppliancesUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateName", func(t *testing.T) {
		dbPath, id := createSeededApplianceDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "appliances", "update", idStr, //nolint:gosec // test binary
			"--name", "Dishwasher Pro",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetAppliance(id)
		require.NoError(t, err)
		assert.Equal(t, "Dishwasher Pro", item.Name)
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		dbPath, id := createSeededApplianceDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "appliances", "update", idStr, //nolint:gosec // test binary
			"--brand", "Miele",
			"--location", "Basement",
			"--cost", "1299.00",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances update multi failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetAppliance(id)
		require.NoError(t, err)
		assert.Equal(t, "Miele", item.Brand)
		assert.Equal(t, "Basement", item.Location)
		require.NotNil(t, item.CostCents)
		assert.Equal(t, int64(129900), *item.CostCents)
	})

	t.Run("ClearOptionalFields", func(t *testing.T) {
		dbPath, id := createSeededApplianceDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "appliances", "update", idStr, //nolint:gosec // test binary
			"--cost", "",
			"--purchase-date", "",
			"--warranty-expiry", "",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "appliances update clear failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		item, err := store.GetAppliance(id)
		require.NoError(t, err)
		assert.Nil(t, item.CostCents)
		assert.Nil(t, item.PurchaseDate)
		assert.Nil(t, item.WarrantyExpiry)
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _ := createSeededApplianceDB(t)
		cmd := exec.Command(bin, "appliances", "update", "9999", //nolint:gosec // test binary
			"--name", "Nope",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestAppliancesDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createSeededApplianceDB(t)
	idStr := idString(id)

	cmd := exec.Command(bin, "appliances", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "appliances delete failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	items, err := store.ListAppliances(false)
	require.NoError(t, err)
	assert.Empty(t, items)

	all, err := store.ListAppliances(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
