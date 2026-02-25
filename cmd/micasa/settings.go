// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"fmt"
	"os"
	"text/tabwriter"
)

type settingsCmd struct {
	Set  settingsSetCmd  `cmd:""                    help:"Set a setting value."`
	List settingsListCmd `cmd:"" default:"withargs" help:"List all settings."`
}

// --- set ---

type settingsSetCmd struct {
	Key    string `arg:"" help:"Setting key (units|currency)."`
	Value  string `arg:"" help:"Setting value."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *settingsSetCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	switch cmd.Key {
	case "units":
		if err := store.PutUnitSystem(cmd.Value); err != nil {
			return err
		}
	case "currency":
		if err := store.PutCurrency(cmd.Value); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown setting %q (want units|currency)", cmd.Key)
	}
	return nil
}

// --- list ---

type settingsListCmd struct {
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *settingsListCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	settings, err := store.ListSettings()
	if err != nil {
		return fmt.Errorf("list settings: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE")
	for _, s := range settings {
		fmt.Fprintf(w, "%s\t%s\n", s[0], s[1])
	}
	return w.Flush()
}
