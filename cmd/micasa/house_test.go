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

// createHouseDB creates a migrated, seeded database with a house profile and
// returns the DB path.
func createHouseDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "house.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.CreateHouseProfile(data.HouseProfile{
		Nickname:     "The Cottage",
		AddressLine1: "123 Elm St",
		City:         "Portland",
		State:        "OR",
		PostalCode:   "97201",
		YearBuilt:    1925,
		SquareFeet:   1800,
		Bedrooms:     3,
		Bathrooms:    1.5,
	}))
	require.NoError(t, store.Close())
	return path
}

// createEmptyHouseDB creates a migrated, seeded database without a house
// profile and returns the DB path.
func createEmptyHouseDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty-house.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.Close())
	return path
}

func TestHouseShow(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Table", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "The Cottage")
		assert.Contains(t, got, "123 Elm St")
		assert.Contains(t, got, "Portland")
		assert.Contains(t, got, "OR")
		assert.Contains(t, got, "97201")
		assert.Contains(t, got, "1925")
		assert.Contains(t, got, "1800")
		assert.Contains(t, got, "1.5")
	})

	t.Run("JSON", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "show", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house show --json failed: %s", out)

		var profile houseProfileJSON
		require.NoError(t, json.Unmarshal(out, &profile))
		assert.Equal(t, "The Cottage", profile.Nickname)
		assert.Equal(t, "123 Elm St", profile.AddressLine1)
		assert.Equal(t, "Portland", profile.City)
		assert.Equal(t, "OR", profile.State)
		assert.Equal(t, "97201", profile.PostalCode)
		assert.Equal(t, 1925, profile.YearBuilt)
		assert.Equal(t, 1800, profile.SquareFeet)
		assert.Equal(t, 3, profile.Bedrooms)
		assert.Equal(t, 1.5, profile.Bathrooms)
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house (default show) failed: %s", out)
		assert.Contains(t, string(out), "The Cottage")
	})

	t.Run("NoProfile", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "no house profile exists")
	})
}

func TestHouseUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("CreateFromScratch", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "New House",
			"--address", "456 Oak Ave",
			"--city", "Seattle",
			"--state", "WA",
			"--postal-code", "98101",
			"--year-built", "2020",
			"--sqft", "2400",
			"--bedrooms", "4",
			"--bathrooms", "2.5",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update (create) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		profile, err := store.HouseProfile()
		require.NoError(t, err)
		assert.Equal(t, "New House", profile.Nickname)
		assert.Equal(t, "456 Oak Ave", profile.AddressLine1)
		assert.Equal(t, "Seattle", profile.City)
		assert.Equal(t, "WA", profile.State)
		assert.Equal(t, "98101", profile.PostalCode)
		assert.Equal(t, 2020, profile.YearBuilt)
		assert.Equal(t, 2400, profile.SquareFeet)
		assert.Equal(t, 4, profile.Bedrooms)
		assert.Equal(t, 2.5, profile.Bathrooms)
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Updated Cottage",
			"--sqft", "2000",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		profile, err := store.HouseProfile()
		require.NoError(t, err)
		assert.Equal(t, "Updated Cottage", profile.Nickname)
		assert.Equal(t, 2000, profile.SquareFeet)
		// Unchanged fields preserved.
		assert.Equal(t, "123 Elm St", profile.AddressLine1)
		assert.Equal(t, "Portland", profile.City)
		assert.Equal(t, 1925, profile.YearBuilt)
	})

	t.Run("UpdateAllFields", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Full House",
			"--address", "789 Pine Rd",
			"--address-line-2", "Unit B",
			"--city", "Denver",
			"--state", "CO",
			"--postal-code", "80202",
			"--year-built", "1990",
			"--sqft", "3000",
			"--lot-sqft", "6000",
			"--bedrooms", "5",
			"--bathrooms", "3",
			"--foundation-type", "slab",
			"--wiring-type", "copper",
			"--roof-type", "asphalt shingle",
			"--exterior-type", "brick",
			"--heating-type", "forced air",
			"--cooling-type", "central AC",
			"--water-source", "municipal",
			"--sewer-type", "municipal",
			"--parking-type", "2-car garage",
			"--basement-type", "finished",
			"--insurance-carrier", "Acme Insurance",
			"--insurance-policy", "POL-123",
			"--insurance-renewal", "2026-06-15",
			"--property-tax-cents", "450000",
			"--hoa-name", "Pine Ridge HOA",
			"--hoa-fee-cents", "35000",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update (all fields) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		p, err := store.HouseProfile()
		require.NoError(t, err)
		assert.Equal(t, "Full House", p.Nickname)
		assert.Equal(t, "789 Pine Rd", p.AddressLine1)
		assert.Equal(t, "Unit B", p.AddressLine2)
		assert.Equal(t, "Denver", p.City)
		assert.Equal(t, "CO", p.State)
		assert.Equal(t, "80202", p.PostalCode)
		assert.Equal(t, 1990, p.YearBuilt)
		assert.Equal(t, 3000, p.SquareFeet)
		assert.Equal(t, 6000, p.LotSquareFeet)
		assert.Equal(t, 5, p.Bedrooms)
		assert.Equal(t, 3.0, p.Bathrooms)
		assert.Equal(t, "slab", p.FoundationType)
		assert.Equal(t, "copper", p.WiringType)
		assert.Equal(t, "asphalt shingle", p.RoofType)
		assert.Equal(t, "brick", p.ExteriorType)
		assert.Equal(t, "forced air", p.HeatingType)
		assert.Equal(t, "central AC", p.CoolingType)
		assert.Equal(t, "municipal", p.WaterSource)
		assert.Equal(t, "municipal", p.SewerType)
		assert.Equal(t, "2-car garage", p.ParkingType)
		assert.Equal(t, "finished", p.BasementType)
		assert.Equal(t, "Acme Insurance", p.InsuranceCarrier)
		assert.Equal(t, "POL-123", p.InsurancePolicy)
		require.NotNil(t, p.InsuranceRenewal)
		assert.Equal(t, "2026-06-15", p.InsuranceRenewal.Format("2006-01-02"))
		require.NotNil(t, p.PropertyTaxCents)
		assert.Equal(t, int64(450000), *p.PropertyTaxCents)
		assert.Equal(t, "Pine Ridge HOA", p.HOAName)
		require.NotNil(t, p.HOAFeeCents)
		assert.Equal(t, int64(35000), *p.HOAFeeCents)
	})

	t.Run("InvalidRenewalDate", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--insurance-renewal", "not-a-date",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid insurance renewal date")
	})

	t.Run("ShowAfterUpdate", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)

		// Create via update.
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Test House",
			"--address", "1 Test Ln",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update failed: %s", out)

		// Verify show works.
		cmd = exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, "house show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Test House")
		assert.Contains(t, got, "1 Test Ln")
	})

	t.Run("ShowJSONMonetaryFields", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Money House",
			"--property-tax-cents", "123456",
			"--hoa-fee-cents", "7890",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update failed: %s", out)

		cmd = exec.Command(bin, "house", "show", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, "house show --json failed: %s", out)

		var profile houseProfileJSON
		require.NoError(t, json.Unmarshal(out, &profile))
		require.NotNil(t, profile.PropertyTaxCents)
		assert.Equal(t, int64(123456), *profile.PropertyTaxCents)
		require.NotNil(t, profile.HOAFeeCents)
		assert.Equal(t, int64(7890), *profile.HOAFeeCents)
	})

	t.Run("ShowTableMonetaryFields", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Tax House",
			"--property-tax-cents", "450000",
			"--hoa-fee-cents", "25000",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update failed: %s", out)

		cmd = exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err = cmd.CombinedOutput()
		require.NoError(t, err, "house show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "$4,500.00")
		assert.Contains(t, got, "$250.00")
	})

	t.Run("NoFlagsCreatesEmpty", func(t *testing.T) {
		dbPath := createEmptyHouseDB(t)
		cmd := exec.Command(bin, "house", "update", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update (no flags) failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))

		// Profile exists but is empty.
		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		_, err = store.HouseProfile()
		require.NoError(t, err)
	})
}
