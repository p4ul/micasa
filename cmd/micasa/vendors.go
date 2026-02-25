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

type vendorsCmd struct {
	List   vendorsListCmd   `cmd:"" default:"withargs" help:"List vendors."`
	Add    vendorsAddCmd    `cmd:""                    help:"Add a new vendor."`
	Show   vendorsShowCmd   `cmd:""                    help:"Show vendor details."`
	Update vendorsUpdateCmd `cmd:""                    help:"Update a vendor."`
	Delete vendorsDeleteCmd `cmd:""                    help:"Delete a vendor (soft-delete)."`
}

type vendorJSON struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ContactName string `json:"contact_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Website     string `json:"website,omitempty"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toVendorJSON(v data.Vendor) vendorJSON {
	return vendorJSON{
		ID:          v.ID,
		Name:        v.Name,
		ContactName: v.ContactName,
		Email:       v.Email,
		Phone:       v.Phone,
		Website:     v.Website,
		Notes:       v.Notes,
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   v.UpdatedAt.Format(time.RFC3339),
	}
}

// --- list ---

type vendorsListCmd struct {
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *vendorsListCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	vendors, err := store.ListVendors(false)
	if err != nil {
		return fmt.Errorf("list vendors: %w", err)
	}

	if cmd.JSON {
		out := make([]vendorJSON, len(vendors))
		for i, v := range vendors {
			out[i] = toVendorJSON(v)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(vendors) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCONTACT\tEMAIL\tPHONE")
	for _, v := range vendors {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			v.ID,
			v.Name,
			v.ContactName,
			v.Email,
			v.Phone,
		)
	}
	return w.Flush()
}

// --- add ---

type vendorsAddCmd struct {
	Name    string `help:"Vendor name."    required:""`
	Contact string `help:"Contact person." name:"contact"`
	Email   string `help:"Email address."`
	Phone   string `help:"Phone number."`
	Website string `help:"Website URL."`
	Notes   string `help:"Additional notes."`
	DBPath  string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *vendorsAddCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	vendor := data.Vendor{
		Name:        cmd.Name,
		ContactName: cmd.Contact,
		Email:       cmd.Email,
		Phone:       cmd.Phone,
		Website:     cmd.Website,
		Notes:       cmd.Notes,
	}
	if err := store.CreateVendor(&vendor); err != nil {
		return fmt.Errorf("create vendor: %w", err)
	}
	fmt.Println(vendor.ID)
	return nil
}

// --- show ---

type vendorsShowCmd struct {
	ID     string `arg:"" help:"Vendor ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *vendorsShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	vendor, err := store.GetVendor(id)
	if err != nil {
		return fmt.Errorf("get vendor %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toVendorJSON(vendor))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", vendor.ID)
	fmt.Fprintf(w, "Name:\t%s\n", vendor.Name)
	if vendor.ContactName != "" {
		fmt.Fprintf(w, "Contact:\t%s\n", vendor.ContactName)
	}
	if vendor.Email != "" {
		fmt.Fprintf(w, "Email:\t%s\n", vendor.Email)
	}
	if vendor.Phone != "" {
		fmt.Fprintf(w, "Phone:\t%s\n", vendor.Phone)
	}
	if vendor.Website != "" {
		fmt.Fprintf(w, "Website:\t%s\n", vendor.Website)
	}
	if vendor.Notes != "" {
		fmt.Fprintf(w, "Notes:\t%s\n", vendor.Notes)
	}
	return w.Flush()
}

// --- update ---

type vendorsUpdateCmd struct {
	ID      string  `arg:"" help:"Vendor ID."`
	Name    *string `help:"New name."`
	Contact *string `help:"New contact person." name:"contact"`
	Email   *string `help:"New email address."`
	Phone   *string `help:"New phone number."`
	Website *string `help:"New website URL."`
	Notes   *string `help:"New notes."`
	DBPath  string  `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *vendorsUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	vendor, err := store.GetVendor(id)
	if err != nil {
		return fmt.Errorf("get vendor %d: %w", id, err)
	}

	if cmd.Name != nil {
		vendor.Name = *cmd.Name
	}
	if cmd.Contact != nil {
		vendor.ContactName = *cmd.Contact
	}
	if cmd.Email != nil {
		vendor.Email = *cmd.Email
	}
	if cmd.Phone != nil {
		vendor.Phone = *cmd.Phone
	}
	if cmd.Website != nil {
		vendor.Website = *cmd.Website
	}
	if cmd.Notes != nil {
		vendor.Notes = *cmd.Notes
	}

	if err := store.UpdateVendor(vendor); err != nil {
		return fmt.Errorf("update vendor %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type vendorsDeleteCmd struct {
	ID     string `arg:"" help:"Vendor ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *vendorsDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteVendor(id); err != nil {
		return fmt.Errorf("delete vendor %d: %w", id, err)
	}
	return nil
}
