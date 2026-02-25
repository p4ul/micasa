// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cpcloud/micasa/internal/data"
)

type maintenanceCmd struct {
	List   maintenanceListCmd   `cmd:"" default:"withargs" help:"List maintenance items."`
	Add    maintenanceAddCmd    `cmd:""                    help:"Add a new maintenance item."`
	Show   maintenanceShowCmd   `cmd:""                    help:"Show maintenance item details."`
	Update maintenanceUpdateCmd `cmd:""                    help:"Update a maintenance item."`
	Delete maintenanceDeleteCmd `cmd:""                    help:"Delete a maintenance item (soft-delete)."`
}

type maintenanceJSON struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	CategoryID     uint    `json:"category_id"`
	ApplianceID    *uint   `json:"appliance_id,omitempty"`
	Appliance      string  `json:"appliance,omitempty"`
	LastServicedAt *string `json:"last_serviced_at,omitempty"`
	IntervalMonths int     `json:"interval_months"`
	CostCents      *int64  `json:"cost_cents,omitempty"`
	Notes          string  `json:"notes,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toMaintenanceJSON(item data.MaintenanceItem) maintenanceJSON {
	j := maintenanceJSON{
		ID:             item.ID,
		Name:           item.Name,
		Category:       item.Category.Name,
		CategoryID:     item.CategoryID,
		ApplianceID:    item.ApplianceID,
		IntervalMonths: item.IntervalMonths,
		CostCents:      item.CostCents,
		Notes:          item.Notes,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
	if item.LastServicedAt != nil {
		s := item.LastServicedAt.Format(time.DateOnly)
		j.LastServicedAt = &s
	}
	if item.Appliance.Name != "" {
		j.Appliance = item.Appliance.Name
	}
	return j
}

func resolveCategory(store *data.Store, name string) (uint, error) {
	cats, err := store.MaintenanceCategories()
	if err != nil {
		return 0, fmt.Errorf("list categories: %w", err)
	}
	names := make([]string, 0, len(cats))
	for _, c := range cats {
		if c.Name == name {
			return c.ID, nil
		}
		names = append(names, c.Name)
	}
	return 0, fmt.Errorf("unknown category %q (want one of: %s)", name, joinNames(names))
}

func joinNames(names []string) string {
	s := ""
	for i, n := range names {
		if i > 0 {
			s += ", "
		}
		s += n
	}
	return s
}

// --- list ---

type maintenanceListCmd struct {
	JSON     bool   `help:"Output as JSON."                                  short:"j"`
	Category string `help:"Filter by category name."                         short:"c"`
	DBPath   string `help:"SQLite database path."              env:"MICASA_DB_PATH"`
}

func (cmd *maintenanceListCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	items, err := store.ListMaintenance(false)
	if err != nil {
		return fmt.Errorf("list maintenance: %w", err)
	}

	if cmd.Category != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.Category.Name == cmd.Category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if cmd.JSON {
		out := make([]maintenanceJSON, len(items))
		for i, item := range items {
			out[i] = toMaintenanceJSON(item)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(items) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tAPPLIANCE\tINTERVAL\tCOST")
	for _, item := range items {
		appliance := ""
		if item.Appliance.Name != "" {
			appliance = item.Appliance.Name
		}
		interval := ""
		if item.IntervalMonths > 0 {
			interval = fmt.Sprintf("%dmo", item.IntervalMonths)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.Category.Name,
			appliance,
			interval,
			data.FormatOptionalCents(item.CostCents),
		)
	}
	return w.Flush()
}

// --- add ---

type maintenanceAddCmd struct {
	Name           string  `help:"Maintenance item name."                     required:""`
	Category       string  `help:"Category name (e.g. HVAC, Plumbing)."      required:""`
	ApplianceID    *uint   `help:"Associated appliance ID."                   name:"appliance-id"`
	IntervalMonths *int    `help:"Service interval in months."                name:"interval-months"`
	Cost           *string `help:"Estimated cost (e.g. 150.00 or $150.00)."  name:"cost"`
	Notes          string  `help:"Additional notes."`
	DBPath         string  `help:"SQLite database path."                      env:"MICASA_DB_PATH"`
}

func (cmd *maintenanceAddCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	catID, err := resolveCategory(store, cmd.Category)
	if err != nil {
		return err
	}

	item := data.MaintenanceItem{
		Name:        cmd.Name,
		CategoryID:  catID,
		ApplianceID: cmd.ApplianceID,
	}
	if cmd.IntervalMonths != nil {
		item.IntervalMonths = *cmd.IntervalMonths
	}
	if cmd.Cost != nil {
		cents, err := data.ParseRequiredCents(*cmd.Cost)
		if err != nil {
			return fmt.Errorf("invalid cost: %w", err)
		}
		item.CostCents = &cents
	}
	if cmd.Notes != "" {
		item.Notes = cmd.Notes
	}
	if err := store.CreateMaintenance(&item); err != nil {
		return fmt.Errorf("create maintenance item: %w", err)
	}
	fmt.Println(item.ID)
	return nil
}

// --- show ---

type maintenanceShowCmd struct {
	ID     string `arg:"" help:"Maintenance item ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *maintenanceShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetMaintenance(id)
	if err != nil {
		return fmt.Errorf("get maintenance item %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toMaintenanceJSON(item))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", item.ID)
	fmt.Fprintf(w, "Name:\t%s\n", item.Name)
	fmt.Fprintf(w, "Category:\t%s\n", item.Category.Name)
	if item.Appliance.Name != "" {
		fmt.Fprintf(w, "Appliance:\t%s (ID %d)\n", item.Appliance.Name, item.Appliance.ID)
	}
	if item.IntervalMonths > 0 {
		fmt.Fprintf(w, "Interval:\t%d months\n", item.IntervalMonths)
	}
	if item.LastServicedAt != nil {
		fmt.Fprintf(w, "Last serviced:\t%s\n", item.LastServicedAt.Format(time.DateOnly))
	}
	if item.CostCents != nil {
		fmt.Fprintf(w, "Cost:\t%s\n", data.FormatCents(*item.CostCents))
	}
	if item.Notes != "" {
		fmt.Fprintf(w, "Notes:\t%s\n", item.Notes)
	}
	return w.Flush()
}

// --- update ---

type maintenanceUpdateCmd struct {
	ID             string  `arg:"" help:"Maintenance item ID."`
	Name           *string `help:"New name."`
	Category       *string `help:"New category name."`
	ApplianceID    *uint   `help:"New appliance ID (0 to clear)." name:"appliance-id"`
	IntervalMonths *int    `help:"New service interval in months." name:"interval-months"`
	Cost           *string `help:"New cost (e.g. 150.00 or $150.00)." name:"cost"`
	Notes          *string `help:"New notes."`
	DBPath         string  `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *maintenanceUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetMaintenance(id)
	if err != nil {
		return fmt.Errorf("get maintenance item %d: %w", id, err)
	}

	if cmd.Name != nil {
		item.Name = *cmd.Name
	}
	if cmd.Category != nil {
		catID, err := resolveCategory(store, *cmd.Category)
		if err != nil {
			return err
		}
		item.CategoryID = catID
	}
	if cmd.ApplianceID != nil {
		if *cmd.ApplianceID == 0 {
			item.ApplianceID = nil
		} else {
			item.ApplianceID = cmd.ApplianceID
		}
	}
	if cmd.IntervalMonths != nil {
		item.IntervalMonths = *cmd.IntervalMonths
	}
	if cmd.Cost != nil {
		cents, err := data.ParseRequiredCents(*cmd.Cost)
		if err != nil {
			return fmt.Errorf("invalid cost: %w", err)
		}
		item.CostCents = &cents
	}
	if cmd.Notes != nil {
		item.Notes = *cmd.Notes
	}

	if err := store.UpdateMaintenance(item); err != nil {
		return fmt.Errorf("update maintenance item %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type maintenanceDeleteCmd struct {
	ID     string `arg:"" help:"Maintenance item ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *maintenanceDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteMaintenance(id); err != nil {
		return fmt.Errorf("delete maintenance item %d: %w", id, err)
	}
	return nil
}
