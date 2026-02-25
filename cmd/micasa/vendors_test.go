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

func createVendorDB(t *testing.T) (string, uint) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vendors.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())

	v := data.Vendor{
		Name:        "Acme Plumbing",
		ContactName: "Jane Doe",
		Email:       "jane@acme.com",
		Phone:       "555-1234",
		Website:     "https://acme.example.com",
	}
	require.NoError(t, store.CreateVendor(&v))
	require.NoError(t, store.Close())
	return path, v.ID
}

func emptyVendorDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty.db")
	store, err := data.Open(path)
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.Close())
	return path
}

func TestVendorsList(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, _ := createVendorDB(t)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "list", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors list failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Acme Plumbing")
		assert.Contains(t, got, "Jane Doe")
		assert.Contains(t, got, "jane@acme.com")
		assert.Contains(t, got, "555-1234")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "list", "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors list --json failed: %s", out)

		var items []vendorJSON
		require.NoError(t, json.Unmarshal(out, &items))
		require.Len(t, items, 1)
		assert.Equal(t, "Acme Plumbing", items[0].Name)
		assert.Equal(t, "Jane Doe", items[0].ContactName)
		assert.Equal(t, "jane@acme.com", items[0].Email)
	})

	t.Run("DefaultSubcommand", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors (default list) failed: %s", out)
		assert.Contains(t, string(out), "Acme Plumbing")
	})

	t.Run("EmptyList", func(t *testing.T) {
		emptyDB := emptyVendorDB(t)
		cmd := exec.Command(bin, "vendors", "list", "--db-path", emptyDB) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors list on empty db failed: %s", out)
		assert.Empty(t, strings.TrimSpace(string(out)))
	})
}

func TestVendorsAdd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("Minimal", func(t *testing.T) {
		dbPath := emptyVendorDB(t)

		cmd := exec.Command(bin, "vendors", "add", //nolint:gosec // test binary
			"--name", "Bob's Electric",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors add failed: %s", out)
		got := strings.TrimSpace(string(out))
		assert.NotEmpty(t, got, "expected vendor ID in output")

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		vendors, err := store.ListVendors(false)
		require.NoError(t, err)
		require.Len(t, vendors, 1)
		assert.Equal(t, "Bob's Electric", vendors[0].Name)
	})

	t.Run("AllFlags", func(t *testing.T) {
		dbPath := emptyVendorDB(t)

		cmd := exec.Command(bin, "vendors", "add", //nolint:gosec // test binary
			"--name", "Pro Roofing",
			"--contact", "Mike Smith",
			"--email", "mike@pro.example.com",
			"--phone", "555-9876",
			"--website", "https://pro.example.com",
			"--notes", "Licensed and insured",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors add (all flags) failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		vendors, err := store.ListVendors(false)
		require.NoError(t, err)
		require.Len(t, vendors, 1)
		assert.Equal(t, "Pro Roofing", vendors[0].Name)
		assert.Equal(t, "Mike Smith", vendors[0].ContactName)
		assert.Equal(t, "mike@pro.example.com", vendors[0].Email)
		assert.Equal(t, "555-9876", vendors[0].Phone)
		assert.Equal(t, "https://pro.example.com", vendors[0].Website)
		assert.Equal(t, "Licensed and insured", vendors[0].Notes)
	})

	t.Run("MissingName", func(t *testing.T) {
		dbPath := emptyVendorDB(t)
		cmd := exec.Command(bin, "vendors", "add", //nolint:gosec // test binary
			"--email", "no-name@example.com",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestVendorsShow(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createVendorDB(t)
	idStr := idString(id)

	t.Run("Table", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "show", idStr, "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors show failed: %s", out)
		got := string(out)
		assert.Contains(t, got, "Acme Plumbing")
		assert.Contains(t, got, "Jane Doe")
		assert.Contains(t, got, "jane@acme.com")
		assert.Contains(t, got, "555-1234")
		assert.Contains(t, got, "https://acme.example.com")
	})

	t.Run("JSON", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "show", idStr, "--json", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors show --json failed: %s", out)

		var item vendorJSON
		require.NoError(t, json.Unmarshal(out, &item))
		assert.Equal(t, "Acme Plumbing", item.Name)
		assert.Equal(t, "Jane Doe", item.ContactName)
		assert.Equal(t, "https://acme.example.com", item.Website)
	})

	t.Run("NotFound", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "show", "9999", "--db-path", dbPath) //nolint:gosec // test binary
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})

	t.Run("InvalidID", func(t *testing.T) {
		cmd := exec.Command(bin, "vendors", "show", "abc", "--db-path", dbPath) //nolint:gosec // test binary
		out, err := cmd.CombinedOutput()
		require.Error(t, err)
		assert.Contains(t, string(out), "invalid ID")
	})
}

func TestVendorsUpdate(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("UpdateName", func(t *testing.T) {
		dbPath, id := createVendorDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "vendors", "update", idStr, //nolint:gosec // test binary
			"--name", "Acme Plumbing Co.",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors update failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		v, err := store.GetVendor(id)
		require.NoError(t, err)
		assert.Equal(t, "Acme Plumbing Co.", v.Name)
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		dbPath, id := createVendorDB(t)
		idStr := idString(id)

		cmd := exec.Command(bin, "vendors", "update", idStr, //nolint:gosec // test binary
			"--email", "new@acme.com",
			"--phone", "555-0000",
			"--notes", "Updated contact info",
			"--db-path", dbPath,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "vendors update multiple fields failed: %s", out)

		store, err := data.Open(dbPath)
		require.NoError(t, err)
		defer func() { _ = store.Close() }()
		v, err := store.GetVendor(id)
		require.NoError(t, err)
		assert.Equal(t, "new@acme.com", v.Email)
		assert.Equal(t, "555-0000", v.Phone)
		assert.Equal(t, "Updated contact info", v.Notes)
		assert.Equal(t, "Acme Plumbing", v.Name, "unchanged fields should be preserved")
	})

	t.Run("NotFound", func(t *testing.T) {
		dbPath, _ := createVendorDB(t)
		cmd := exec.Command(bin, "vendors", "update", "9999", //nolint:gosec // test binary
			"--name", "Nope",
			"--db-path", dbPath,
		)
		_, err := cmd.CombinedOutput()
		require.Error(t, err)
	})
}

func TestVendorsDelete(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createVendorDB(t)
	idStr := idString(id)

	cmd := exec.Command(bin, "vendors", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "vendors delete failed: %s", out)

	store, err := data.Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	vendors, err := store.ListVendors(false)
	require.NoError(t, err)
	assert.Empty(t, vendors)

	all, err := store.ListVendors(true)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestVendorsDeleteWithDependents(t *testing.T) {
	bin := buildTestBinary(t)
	dbPath, id := createVendorDB(t)
	idStr := idString(id)

	// Create an incident referencing this vendor so delete is blocked.
	store, err := data.Open(dbPath)
	require.NoError(t, err)
	inc := data.Incident{
		Title:    "Test incident",
		Status:   data.IncidentStatusOpen,
		Severity: data.IncidentSeverityWhenever,
		VendorID: &id,
	}
	require.NoError(t, store.CreateIncident(&inc))
	require.NoError(t, store.Close())

	cmd := exec.Command(bin, "vendors", "delete", idStr, "--db-path", dbPath) //nolint:gosec // test binary
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "active incident")
}
