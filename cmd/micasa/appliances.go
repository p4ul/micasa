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

type appliancesCmd struct {
	List   appliancesListCmd   `cmd:"" default:"withargs" help:"List appliances."`
	Add    appliancesAddCmd    `cmd:""                    help:"Add a new appliance."`
	Show   appliancesShowCmd   `cmd:""                    help:"Show appliance details."`
	Update appliancesUpdateCmd `cmd:""                    help:"Update an appliance."`
	Delete appliancesDeleteCmd `cmd:""                    help:"Delete an appliance (soft-delete)."`
}

type applianceJSON struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Brand          string `json:"brand,omitempty"`
	ModelNumber    string `json:"model_number,omitempty"`
	SerialNumber   string `json:"serial_number,omitempty"`
	PurchaseDate   string `json:"purchase_date,omitempty"`
	WarrantyExpiry string `json:"warranty_expiry,omitempty"`
	Location       string `json:"location,omitempty"`
	CostCents      *int64 `json:"cost_cents,omitempty"`
	Notes          string `json:"notes,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func toApplianceJSON(item data.Appliance) applianceJSON {
	j := applianceJSON{
		ID:           item.ID,
		Name:         item.Name,
		Brand:        item.Brand,
		ModelNumber:  item.ModelNumber,
		SerialNumber: item.SerialNumber,
		Location:     item.Location,
		CostCents:    item.CostCents,
		Notes:        item.Notes,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
	if item.PurchaseDate != nil {
		j.PurchaseDate = item.PurchaseDate.Format(time.DateOnly)
	}
	if item.WarrantyExpiry != nil {
		j.WarrantyExpiry = item.WarrantyExpiry.Format(time.DateOnly)
	}
	return j
}

// --- list ---

type appliancesListCmd struct {
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *appliancesListCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	items, err := store.ListAppliances(false)
	if err != nil {
		return fmt.Errorf("list appliances: %w", err)
	}

	if cmd.JSON {
		out := make([]applianceJSON, len(items))
		for i, item := range items {
			out[i] = toApplianceJSON(item)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(items) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tBRAND\tMODEL\tLOCATION\tWARRANTY")
	for _, item := range items {
		warranty := ""
		if item.WarrantyExpiry != nil {
			warranty = item.WarrantyExpiry.Format(time.DateOnly)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.Brand,
			item.ModelNumber,
			item.Location,
			warranty,
		)
	}
	return w.Flush()
}

// --- add ---

type appliancesAddCmd struct {
	Name           string `help:"Appliance name."                        required:""`
	Brand          string `help:"Brand or manufacturer."`
	Model          string `help:"Model number."                          name:"model"`
	Serial         string `help:"Serial number."                         name:"serial"`
	Location       string `help:"Location in the home."`
	Cost           string `help:"Purchase cost (e.g. 499.99 or $499.99)." name:"cost"`
	PurchaseDate   string `help:"Purchase date (YYYY-MM-DD)."            name:"purchase-date"`
	WarrantyExpiry string `help:"Warranty expiry date (YYYY-MM-DD)."     name:"warranty-expiry"`
	Notes          string `help:"Additional notes."`
	DBPath         string `help:"SQLite database path."                  env:"MICASA_DB_PATH"`
}

func (cmd *appliancesAddCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item := data.Appliance{
		Name:         cmd.Name,
		Brand:        cmd.Brand,
		ModelNumber:  cmd.Model,
		SerialNumber: cmd.Serial,
		Location:     cmd.Location,
		Notes:        cmd.Notes,
	}

	if cmd.Cost != "" {
		cents, err := data.ParseRequiredCents(cmd.Cost)
		if err != nil {
			return fmt.Errorf("parse cost: %w", err)
		}
		item.CostCents = &cents
	}
	if cmd.PurchaseDate != "" {
		t, err := time.Parse(time.DateOnly, cmd.PurchaseDate)
		if err != nil {
			return fmt.Errorf("parse purchase date: expected YYYY-MM-DD, got %q", cmd.PurchaseDate)
		}
		item.PurchaseDate = &t
	}
	if cmd.WarrantyExpiry != "" {
		t, err := time.Parse(time.DateOnly, cmd.WarrantyExpiry)
		if err != nil {
			return fmt.Errorf("parse warranty expiry: expected YYYY-MM-DD, got %q", cmd.WarrantyExpiry)
		}
		item.WarrantyExpiry = &t
	}

	if err := store.CreateAppliance(&item); err != nil {
		return fmt.Errorf("create appliance: %w", err)
	}
	fmt.Println(item.ID)
	return nil
}

// --- show ---

type appliancesShowCmd struct {
	ID     string `arg:"" help:"Appliance ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *appliancesShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetAppliance(id)
	if err != nil {
		return fmt.Errorf("get appliance %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toApplianceJSON(item))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", item.ID)
	fmt.Fprintf(w, "Name:\t%s\n", item.Name)
	if item.Brand != "" {
		fmt.Fprintf(w, "Brand:\t%s\n", item.Brand)
	}
	if item.ModelNumber != "" {
		fmt.Fprintf(w, "Model:\t%s\n", item.ModelNumber)
	}
	if item.SerialNumber != "" {
		fmt.Fprintf(w, "Serial:\t%s\n", item.SerialNumber)
	}
	if item.Location != "" {
		fmt.Fprintf(w, "Location:\t%s\n", item.Location)
	}
	if item.PurchaseDate != nil {
		fmt.Fprintf(w, "Purchased:\t%s\n", item.PurchaseDate.Format(time.DateOnly))
	}
	if item.WarrantyExpiry != nil {
		fmt.Fprintf(w, "Warranty:\t%s\n", item.WarrantyExpiry.Format(time.DateOnly))
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

type appliancesUpdateCmd struct {
	ID             string  `arg:"" help:"Appliance ID."`
	Name           *string `help:"New name."`
	Brand          *string `help:"New brand."`
	Model          *string `help:"New model number."                      name:"model"`
	Serial         *string `help:"New serial number."                     name:"serial"`
	Location       *string `help:"New location."`
	Cost           *string `help:"New cost (e.g. 499.99; empty to clear)." name:"cost"`
	PurchaseDate   *string `help:"New purchase date (YYYY-MM-DD; empty to clear)." name:"purchase-date"`
	WarrantyExpiry *string `help:"New warranty expiry (YYYY-MM-DD; empty to clear)." name:"warranty-expiry"`
	Notes          *string `help:"New notes."`
	DBPath         string  `help:"SQLite database path."                  env:"MICASA_DB_PATH"`
}

func (cmd *appliancesUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetAppliance(id)
	if err != nil {
		return fmt.Errorf("get appliance %d: %w", id, err)
	}

	if cmd.Name != nil {
		item.Name = *cmd.Name
	}
	if cmd.Brand != nil {
		item.Brand = *cmd.Brand
	}
	if cmd.Model != nil {
		item.ModelNumber = *cmd.Model
	}
	if cmd.Serial != nil {
		item.SerialNumber = *cmd.Serial
	}
	if cmd.Location != nil {
		item.Location = *cmd.Location
	}
	if cmd.Notes != nil {
		item.Notes = *cmd.Notes
	}
	if cmd.Cost != nil {
		if *cmd.Cost == "" {
			item.CostCents = nil
		} else {
			cents, err := data.ParseRequiredCents(*cmd.Cost)
			if err != nil {
				return fmt.Errorf("parse cost: %w", err)
			}
			item.CostCents = &cents
		}
	}
	if cmd.PurchaseDate != nil {
		if *cmd.PurchaseDate == "" {
			item.PurchaseDate = nil
		} else {
			t, err := time.Parse(time.DateOnly, *cmd.PurchaseDate)
			if err != nil {
				return fmt.Errorf("parse purchase date: expected YYYY-MM-DD, got %q", *cmd.PurchaseDate)
			}
			item.PurchaseDate = &t
		}
	}
	if cmd.WarrantyExpiry != nil {
		if *cmd.WarrantyExpiry == "" {
			item.WarrantyExpiry = nil
		} else {
			t, err := time.Parse(time.DateOnly, *cmd.WarrantyExpiry)
			if err != nil {
				return fmt.Errorf("parse warranty expiry: expected YYYY-MM-DD, got %q", *cmd.WarrantyExpiry)
			}
			item.WarrantyExpiry = &t
		}
	}

	if err := store.UpdateAppliance(item); err != nil {
		return fmt.Errorf("update appliance %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type appliancesDeleteCmd struct {
	ID     string `arg:"" help:"Appliance ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *appliancesDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteAppliance(id); err != nil {
		return fmt.Errorf("delete appliance %d: %w", id, err)
	}
	return nil
}
