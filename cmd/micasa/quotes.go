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

type quotesCmd struct {
	List   quotesListCmd   `cmd:"" default:"withargs" help:"List quotes."`
	Add    quotesAddCmd    `cmd:""                    help:"Add a new quote."`
	Show   quotesShowCmd   `cmd:""                    help:"Show quote details."`
	Update quotesUpdateCmd `cmd:""                    help:"Update a quote."`
	Delete quotesDeleteCmd `cmd:""                    help:"Delete a quote (soft-delete)."`
}

type quoteJSON struct {
	ID             uint   `json:"id"`
	ProjectID      uint   `json:"project_id"`
	Project        string `json:"project"`
	VendorID       uint   `json:"vendor_id"`
	Vendor         string `json:"vendor"`
	TotalCents     int64  `json:"total_cents"`
	LaborCents     *int64 `json:"labor_cents,omitempty"`
	MaterialsCents *int64 `json:"materials_cents,omitempty"`
	OtherCents     *int64 `json:"other_cents,omitempty"`
	ReceivedDate   string `json:"received_date,omitempty"`
	Notes          string `json:"notes,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func toQuoteJSON(q data.Quote) quoteJSON {
	j := quoteJSON{
		ID:             q.ID,
		ProjectID:      q.ProjectID,
		Project:        q.Project.Title,
		VendorID:       q.VendorID,
		Vendor:         q.Vendor.Name,
		TotalCents:     q.TotalCents,
		LaborCents:     q.LaborCents,
		MaterialsCents: q.MaterialsCents,
		OtherCents:     q.OtherCents,
		Notes:          q.Notes,
		CreatedAt:      q.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      q.UpdatedAt.Format(time.RFC3339),
	}
	if q.ReceivedDate != nil {
		j.ReceivedDate = q.ReceivedDate.Format(time.DateOnly)
	}
	return j
}

// --- list ---

type quotesListCmd struct {
	JSON      bool   `help:"Output as JSON."                 short:"j"`
	ProjectID *uint  `help:"Filter by project ID."           name:"project-id"`
	VendorID  *uint  `help:"Filter by vendor ID."            name:"vendor-id"`
	DBPath    string `help:"SQLite database path."           env:"MICASA_DB_PATH"`
}

func (cmd *quotesListCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	var items []data.Quote
	switch {
	case cmd.ProjectID != nil:
		items, err = store.ListQuotesByProject(*cmd.ProjectID, false)
	case cmd.VendorID != nil:
		items, err = store.ListQuotesByVendor(*cmd.VendorID, false)
	default:
		items, err = store.ListQuotes(false)
	}
	if err != nil {
		return fmt.Errorf("list quotes: %w", err)
	}

	if cmd.JSON {
		out := make([]quoteJSON, len(items))
		for i, item := range items {
			out[i] = toQuoteJSON(item)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(items) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROJECT\tVENDOR\tTOTAL\tRECEIVED")
	for _, item := range items {
		received := ""
		if item.ReceivedDate != nil {
			received = item.ReceivedDate.Format(time.DateOnly)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Project.Title,
			item.Vendor.Name,
			data.FormatCents(item.TotalCents),
			received,
		)
	}
	return w.Flush()
}

// --- add ---

type quotesAddCmd struct {
	ProjectID    uint   `help:"Project ID."                      required:"" name:"project-id"`
	VendorID     uint   `help:"Vendor ID."                       required:"" name:"vendor-id"`
	Total        string `help:"Total amount (e.g. 1500.00)."     required:""`
	Labor        string `help:"Labor amount (e.g. 800.00)."`
	Materials    string `help:"Materials amount (e.g. 500.00)."`
	Other        string `help:"Other amount (e.g. 200.00)."`
	ReceivedDate string `help:"Date quote was received (YYYY-MM-DD)." name:"received-date"`
	Notes        string `help:"Additional notes."`
	DBPath       string `help:"SQLite database path."            env:"MICASA_DB_PATH"`
}

func (cmd *quotesAddCmd) Run() error {
	totalCents, err := data.ParseRequiredCents(cmd.Total)
	if err != nil {
		return fmt.Errorf("parse total: %w", err)
	}
	laborCents, err := data.ParseOptionalCents(cmd.Labor)
	if err != nil {
		return fmt.Errorf("parse labor: %w", err)
	}
	materialsCents, err := data.ParseOptionalCents(cmd.Materials)
	if err != nil {
		return fmt.Errorf("parse materials: %w", err)
	}
	otherCents, err := data.ParseOptionalCents(cmd.Other)
	if err != nil {
		return fmt.Errorf("parse other: %w", err)
	}

	var receivedDate *time.Time
	if cmd.ReceivedDate != "" {
		t, err := time.Parse(time.DateOnly, cmd.ReceivedDate)
		if err != nil {
			return fmt.Errorf("parse received-date: expected YYYY-MM-DD, got %q", cmd.ReceivedDate)
		}
		receivedDate = &t
	}

	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	vendor, err := store.GetVendor(cmd.VendorID)
	if err != nil {
		return fmt.Errorf("get vendor %d: %w", cmd.VendorID, err)
	}

	quote := data.Quote{
		ProjectID:      cmd.ProjectID,
		TotalCents:     totalCents,
		LaborCents:     laborCents,
		MaterialsCents: materialsCents,
		OtherCents:     otherCents,
		ReceivedDate:   receivedDate,
		Notes:          cmd.Notes,
	}
	if err := store.CreateQuote(&quote, vendor); err != nil {
		return fmt.Errorf("create quote: %w", err)
	}
	fmt.Println(quote.ID)
	return nil
}

// --- show ---

type quotesShowCmd struct {
	ID     string `arg:"" help:"Quote ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *quotesShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	quote, err := store.GetQuote(id)
	if err != nil {
		return fmt.Errorf("get quote %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toQuoteJSON(quote))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", quote.ID)
	fmt.Fprintf(w, "Project:\t%s (ID %d)\n", quote.Project.Title, quote.ProjectID)
	fmt.Fprintf(w, "Vendor:\t%s (ID %d)\n", quote.Vendor.Name, quote.VendorID)
	fmt.Fprintf(w, "Total:\t%s\n", data.FormatCents(quote.TotalCents))
	if quote.LaborCents != nil {
		fmt.Fprintf(w, "Labor:\t%s\n", data.FormatCents(*quote.LaborCents))
	}
	if quote.MaterialsCents != nil {
		fmt.Fprintf(w, "Materials:\t%s\n", data.FormatCents(*quote.MaterialsCents))
	}
	if quote.OtherCents != nil {
		fmt.Fprintf(w, "Other:\t%s\n", data.FormatCents(*quote.OtherCents))
	}
	if quote.ReceivedDate != nil {
		fmt.Fprintf(w, "Received:\t%s\n", quote.ReceivedDate.Format(time.DateOnly))
	}
	if quote.Notes != "" {
		fmt.Fprintf(w, "Notes:\t%s\n", quote.Notes)
	}
	return w.Flush()
}

// --- update ---

type quotesUpdateCmd struct {
	ID           string  `arg:"" help:"Quote ID."`
	ProjectID    *uint   `help:"New project ID."                          name:"project-id"`
	VendorID     *uint   `help:"New vendor ID."                           name:"vendor-id"`
	Total        *string `help:"New total amount (e.g. 1500.00)."`
	Labor        *string `help:"New labor amount (e.g. 800.00, empty to clear)."`
	Materials    *string `help:"New materials amount (e.g. 500.00, empty to clear)."`
	Other        *string `help:"New other amount (e.g. 200.00, empty to clear)."`
	ReceivedDate *string `help:"New received date (YYYY-MM-DD, empty to clear)." name:"received-date"`
	Notes        *string `help:"New notes."`
	DBPath       string  `help:"SQLite database path."                    env:"MICASA_DB_PATH"`
}

func (cmd *quotesUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	quote, err := store.GetQuote(id)
	if err != nil {
		return fmt.Errorf("get quote %d: %w", id, err)
	}

	if cmd.ProjectID != nil {
		quote.ProjectID = *cmd.ProjectID
	}
	if cmd.Total != nil {
		cents, err := data.ParseRequiredCents(*cmd.Total)
		if err != nil {
			return fmt.Errorf("parse total: %w", err)
		}
		quote.TotalCents = cents
	}
	if cmd.Labor != nil {
		cents, err := data.ParseOptionalCents(*cmd.Labor)
		if err != nil {
			return fmt.Errorf("parse labor: %w", err)
		}
		quote.LaborCents = cents
	}
	if cmd.Materials != nil {
		cents, err := data.ParseOptionalCents(*cmd.Materials)
		if err != nil {
			return fmt.Errorf("parse materials: %w", err)
		}
		quote.MaterialsCents = cents
	}
	if cmd.Other != nil {
		cents, err := data.ParseOptionalCents(*cmd.Other)
		if err != nil {
			return fmt.Errorf("parse other: %w", err)
		}
		quote.OtherCents = cents
	}
	if cmd.ReceivedDate != nil {
		if *cmd.ReceivedDate == "" {
			quote.ReceivedDate = nil
		} else {
			t, err := time.Parse(time.DateOnly, *cmd.ReceivedDate)
			if err != nil {
				return fmt.Errorf("parse received-date: expected YYYY-MM-DD, got %q", *cmd.ReceivedDate)
			}
			quote.ReceivedDate = &t
		}
	}
	if cmd.Notes != nil {
		quote.Notes = *cmd.Notes
	}

	vendor := quote.Vendor
	if cmd.VendorID != nil {
		vendor, err = store.GetVendor(*cmd.VendorID)
		if err != nil {
			return fmt.Errorf("get vendor %d: %w", *cmd.VendorID, err)
		}
	}

	if err := store.UpdateQuote(quote, vendor); err != nil {
		return fmt.Errorf("update quote %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type quotesDeleteCmd struct {
	ID     string `arg:"" help:"Quote ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *quotesDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteQuote(id); err != nil {
		return fmt.Errorf("delete quote %d: %w", id, err)
	}
	return nil
}
