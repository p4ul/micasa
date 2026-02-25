// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cpcloud/micasa/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createHouseDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "house.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.Close())
	return path
}

func createHouseDBWithProfile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "house.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.CreateHouseProfile(data.HouseProfile{
		Nickname:     "Main House",
		AddressLine1: "123 Main St",
		City:         "Portland",
		State:        "OR",
		PostalCode:   "97201",
		YearBuilt:    1925,
		SquareFeet:   2000,
		Bedrooms:     3,
		Bathrooms:    2.5,
	}))
	require.NoError(t, store.Close())
	return path
}

func TestHouseShow(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Table", func(t *testing.T) {
		dbPath := createHouseDBWithProfile(t)
		cmd := exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Main House")
		assert.Contains(t, got, "123 Main St")
		assert.Contains(t, got, "Portland")
		assert.Contains(t, got, "OR")
		assert.Contains(t, got, "97201")
		assert.Contains(t, got, "1925")
		assert.Contains(t, got, "2000")
		assert.Contains(t, got, "3")
		assert.Contains(t, got, "2.5")
	})

	t.Run("JSON", func(t *testing.T) {
		dbPath := createHouseDBWithProfile(t)
		cmd := exec.Command(bin, "house", "show", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house show --json failed: %s", out)

		var profile houseProfileJSON
		require.NoError(t, json.Unmarshal(out, &profile))
		assert.Equal(t, "Main House", profile.Nickname)
		assert.Equal(t, "123 Main St", profile.AddressLine1)
		assert.Equal(t, "Portland", profile.City)
		assert.Equal(t, "OR", profile.State)
		assert.Equal(t, 1925, profile.YearBuilt)
		assert.Equal(t, 2000, profile.SquareFeet)
		assert.Equal(t, 3, profile.Bedrooms)
		assert.InDelta(t, 2.5, profile.Bathrooms, 0.01)
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		dbPath := createHouseDBWithProfile(t)
		cmd := exec.Command(bin, "house", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house (default show) failed: %s", out)
		assert.Contains(t, string(out), "Main House")
	})

	t.Run("NoProfile", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "show", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "no house profile configured")
	})
}

func TestHouseUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("CreateFromScratch", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Beach House",
			"--address", "456 Ocean Ave",
			"--city", "Malibu",
			"--state", "CA",
			"--postal-code", "90265",
			"--year-built", "1985",
			"--sqft", "3200",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update (create) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		profile, err := store.HouseProfile()
		require.NoError(t, err)
		assert.Equal(t, "Beach House", profile.Nickname)
		assert.Equal(t, "456 Ocean Ave", profile.AddressLine1)
		assert.Equal(t, "Malibu", profile.City)
		assert.Equal(t, "CA", profile.State)
		assert.Equal(t, "90265", profile.PostalCode)
		assert.Equal(t, 1985, profile.YearBuilt)
		assert.Equal(t, 3200, profile.SquareFeet)
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		dbPath := createHouseDBWithProfile(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--city", "Seattle",
			"--state", "WA",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "house update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		profile, err := store.HouseProfile()
		require.NoError(t, err)
		assert.Equal(t, "Seattle", profile.City)
		assert.Equal(t, "WA", profile.State)
		// Unchanged fields preserved.
		assert.Equal(t, "Main House", profile.Nickname)
		assert.Equal(t, "123 Main St", profile.AddressLine1)
		assert.Equal(t, 1925, profile.YearBuilt)
	})

	t.Run("AllFields", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--nickname", "Full House",
			"--address", "789 Full St",
			"--address-line-2", "Unit B",
			"--city", "Chicago",
			"--state", "IL",
			"--postal-code", "60601",
			"--year-built", "2000",
			"--sqft", "1500",
			"--lot-sqft", "5000",
			"--bedrooms", "4",
			"--bathrooms", "3.5",
			"--foundation-type", "slab",
			"--wiring-type", "copper",
			"--roof-type", "asphalt",
			"--exterior-type", "brick",
			"--heating-type", "forced air",
			"--cooling-type", "central",
			"--water-source", "municipal",
			"--sewer-type", "public",
			"--parking-type", "garage",
			"--basement-type", "finished",
			"--insurance-carrier", "State Farm",
			"--insurance-policy", "POL-12345",
			"--insurance-renewal", "2026-06-15",
			"--property-tax-cents", "450000",
			"--hoa-name", "Oak Park HOA",
			"--hoa-fee-cents", "25000",
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
		assert.Equal(t, "789 Full St", p.AddressLine1)
		assert.Equal(t, "Unit B", p.AddressLine2)
		assert.Equal(t, "Chicago", p.City)
		assert.Equal(t, "IL", p.State)
		assert.Equal(t, "60601", p.PostalCode)
		assert.Equal(t, 2000, p.YearBuilt)
		assert.Equal(t, 1500, p.SquareFeet)
		assert.Equal(t, 5000, p.LotSquareFeet)
		assert.Equal(t, 4, p.Bedrooms)
		assert.InDelta(t, 3.5, p.Bathrooms, 0.01)
		assert.Equal(t, "slab", p.FoundationType)
		assert.Equal(t, "copper", p.WiringType)
		assert.Equal(t, "asphalt", p.RoofType)
		assert.Equal(t, "brick", p.ExteriorType)
		assert.Equal(t, "forced air", p.HeatingType)
		assert.Equal(t, "central", p.CoolingType)
		assert.Equal(t, "municipal", p.WaterSource)
		assert.Equal(t, "public", p.SewerType)
		assert.Equal(t, "garage", p.ParkingType)
		assert.Equal(t, "finished", p.BasementType)
		assert.Equal(t, "State Farm", p.InsuranceCarrier)
		assert.Equal(t, "POL-12345", p.InsurancePolicy)
		require.NotNil(t, p.InsuranceRenewal)
		assert.Equal(t, "2026-06-15", p.InsuranceRenewal.Format("2006-01-02"))
		require.NotNil(t, p.PropertyTaxCents)
		assert.Equal(t, int64(450000), *p.PropertyTaxCents)
		assert.Equal(t, "Oak Park HOA", p.HOAName)
		require.NotNil(t, p.HOAFeeCents)
		assert.Equal(t, int64(25000), *p.HOAFeeCents)
	})

	t.Run("InvalidRenewalDate", func(t *testing.T) {
		dbPath := createHouseDB(t)
		cmd := exec.Command(bin, "house", "update", //nolint:gosec // test binary
			"--insurance-renewal", "not-a-date",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid insurance renewal date")
	})
}
