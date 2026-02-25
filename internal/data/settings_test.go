// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSettingMissing(t *testing.T) {
	store := newTestStore(t)
	val, err := store.GetSetting("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestPutAndGetSetting(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.PutSetting("color", "blue"))

	val, err := store.GetSetting("color")
	require.NoError(t, err)
	assert.Equal(t, "blue", val)
}

func TestPutSettingUpserts(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.PutSetting("color", "blue"))
	require.NoError(t, store.PutSetting("color", "red"))

	val, err := store.GetSetting("color")
	require.NoError(t, err)
	assert.Equal(t, "red", val)
}

func TestLastModelRoundTrip(t *testing.T) {
	store := newTestStore(t)

	// Initially empty.
	model, err := store.GetLastModel()
	require.NoError(t, err)
	assert.Equal(t, "", model)

	// Set and retrieve.
	require.NoError(t, store.PutLastModel("qwen3:8b"))
	model, err = store.GetLastModel()
	require.NoError(t, err)
	assert.Equal(t, "qwen3:8b", model)

	// Overwrite.
	require.NoError(t, store.PutLastModel("llama3.3"))
	model, err = store.GetLastModel()
	require.NoError(t, err)
	assert.Equal(t, "llama3.3", model)
}

func TestAppendChatInputAndLoad(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.AppendChatInput("how many projects?"))
	require.NoError(t, store.AppendChatInput("oldest appliance?"))

	history, err := store.LoadChatHistory()
	require.NoError(t, err)
	assert.Equal(t, []string{"how many projects?", "oldest appliance?"}, history)
}

func TestAppendChatInputDeduplicatesConsecutive(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.AppendChatInput("hello"))
	require.NoError(t, store.AppendChatInput("hello"))
	require.NoError(t, store.AppendChatInput("hello"))

	history, err := store.LoadChatHistory()
	require.NoError(t, err)
	assert.Equal(t, []string{"hello"}, history)
}

func TestAppendChatInputAllowsNonConsecutiveDuplicates(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.AppendChatInput("a"))
	require.NoError(t, store.AppendChatInput("b"))
	require.NoError(t, store.AppendChatInput("a"))

	history, err := store.LoadChatHistory()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "a"}, history)
}

func TestLoadChatHistoryEmpty(t *testing.T) {
	store := newTestStore(t)

	history, err := store.LoadChatHistory()
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestGetUnitSystemDefault(t *testing.T) {
	store := newTestStore(t)
	val, err := store.GetUnitSystem()
	require.NoError(t, err)
	assert.Equal(t, DefaultUnitSystem, val)
}

func TestPutAndGetUnitSystem(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.PutUnitSystem("imperial"))
	val, err := store.GetUnitSystem()
	require.NoError(t, err)
	assert.Equal(t, "imperial", val)

	require.NoError(t, store.PutUnitSystem("metric"))
	val, err = store.GetUnitSystem()
	require.NoError(t, err)
	assert.Equal(t, "metric", val)
}

func TestPutUnitSystemInvalid(t *testing.T) {
	store := newTestStore(t)
	err := store.PutUnitSystem("cubits")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid unit system")
}

func TestGetCurrencyDefault(t *testing.T) {
	store := newTestStore(t)
	val, err := store.GetCurrency()
	require.NoError(t, err)
	assert.Equal(t, DefaultCurrency, val)
}

func TestPutAndGetCurrency(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.PutCurrency("USD"))
	val, err := store.GetCurrency()
	require.NoError(t, err)
	assert.Equal(t, "USD", val)

	require.NoError(t, store.PutCurrency("eur"))
	val, err = store.GetCurrency()
	require.NoError(t, err)
	assert.Equal(t, "EUR", val, "should store uppercase")
}

func TestPutCurrencyInvalid(t *testing.T) {
	store := newTestStore(t)
	err := store.PutCurrency("DOGECOIN")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid currency")
}

func TestListSettings(t *testing.T) {
	store := newTestStore(t)

	// Defaults when nothing is set.
	settings, err := store.ListSettings()
	require.NoError(t, err)
	require.Len(t, settings, 2)
	assert.Equal(t, "units.system", settings[0][0])
	assert.Equal(t, DefaultUnitSystem, settings[0][1])
	assert.Equal(t, "units.currency", settings[1][0])
	assert.Equal(t, DefaultCurrency, settings[1][1])

	// After setting values.
	require.NoError(t, store.PutUnitSystem("imperial"))
	require.NoError(t, store.PutCurrency("USD"))
	settings, err = store.ListSettings()
	require.NoError(t, err)
	assert.Equal(t, "imperial", settings[0][1])
	assert.Equal(t, "USD", settings[1][1])
}

func TestValidateUnitSystem(t *testing.T) {
	assert.NoError(t, ValidateUnitSystem("metric"))
	assert.NoError(t, ValidateUnitSystem("imperial"))
	assert.Error(t, ValidateUnitSystem("cubits"))
	assert.Error(t, ValidateUnitSystem(""))
}

func TestValidateCurrency(t *testing.T) {
	for _, code := range ValidCurrencies {
		assert.NoError(t, ValidateCurrency(code))
	}
	assert.NoError(t, ValidateCurrency("nzd"), "should accept lowercase")
	assert.Error(t, ValidateCurrency("DOGECOIN"))
	assert.Error(t, ValidateCurrency(""))
}

func TestShowDashboardDefaultsToTrue(t *testing.T) {
	store := newTestStore(t)
	show, err := store.GetShowDashboard()
	require.NoError(t, err)
	assert.True(t, show, "should default to true when no preference saved")
}

func TestShowDashboardRoundTrip(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.PutShowDashboard(false))
	show, err := store.GetShowDashboard()
	require.NoError(t, err)
	assert.False(t, show)

	require.NoError(t, store.PutShowDashboard(true))
	show, err = store.GetShowDashboard()
	require.NoError(t, err)
	assert.True(t, show)
}
