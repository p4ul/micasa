// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cpcloud/micasa/internal/data"
)

type incidentsCmd struct {
	List    incidentsListCmd    `cmd:"" default:"withargs" help:"List incidents."`
	Add     incidentsAddCmd     `cmd:""                    help:"Add a new incident."`
	Show    incidentsShowCmd    `cmd:""                    help:"Show incident details."`
	Update  incidentsUpdateCmd  `cmd:""                    help:"Update an incident."`
	Delete  incidentsDeleteCmd  `cmd:""                    help:"Delete an incident (soft-delete)."`
	Resolve incidentsResolveCmd `cmd:""                    help:"Resolve an incident."`
}

// incidentJSON is the JSON representation of an incident.
type incidentJSON struct {
	ID           uint    `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description,omitempty"`
	Status       string  `json:"status"`
	Severity     string  `json:"severity"`
	DateNoticed  string  `json:"date_noticed"`
	DateResolved *string `json:"date_resolved,omitempty"`
	Location     string  `json:"location,omitempty"`
	CostCents    *int64  `json:"cost_cents,omitempty"`
	ApplianceID  *uint   `json:"appliance_id,omitempty"`
	Appliance    string  `json:"appliance,omitempty"`
	VendorID     *uint   `json:"vendor_id,omitempty"`
	Vendor       string  `json:"vendor,omitempty"`
	Notes        string  `json:"notes,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toIncidentJSON(item data.Incident) incidentJSON {
	j := incidentJSON{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Status:      item.Status,
		Severity:    item.Severity,
		DateNoticed: item.DateNoticed.Format(time.DateOnly),
		Location:    item.Location,
		CostCents:   item.CostCents,
		ApplianceID: item.ApplianceID,
		VendorID:    item.VendorID,
		Notes:       item.Notes,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
	if item.DateResolved != nil {
		s := item.DateResolved.Format(time.DateOnly)
		j.DateResolved = &s
	}
	if item.Appliance.Name != "" {
		j.Appliance = item.Appliance.Name
	}
	if item.Vendor.Name != "" {
		j.Vendor = item.Vendor.Name
	}
	return j
}

func openStore(dbPath string) (*data.Store, error) {
	if dbPath == "" {
		var err error
		dbPath, err = data.DefaultDBPath()
		if err != nil {
			return nil, fmt.Errorf("resolve db path: %w", err)
		}
	} else {
		dbPath = data.ExpandHome(dbPath)
	}
	store, err := data.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return store, nil
}

var validStatuses = []string{
	data.IncidentStatusOpen,
	data.IncidentStatusInProgress,
	data.IncidentStatusResolved,
}

var validSeverities = []string{
	data.IncidentSeverityUrgent,
	data.IncidentSeveritySoon,
	data.IncidentSeverityWhenever,
}

func validateStatus(s string) error {
	for _, v := range validStatuses {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q (want %s)", s, strings.Join(validStatuses, "|"))
}

func validateSeverity(s string) error {
	for _, v := range validSeverities {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("invalid severity %q (want %s)", s, strings.Join(validSeverities, "|"))
}

// --- list ---

type incidentsListCmd struct {
	JSON   bool   `help:"Output as JSON."                                            short:"j"`
	Status string `help:"Filter by status (open|in_progress|resolved)."              short:"s"`
	DBPath string `help:"SQLite database path."                        env:"MICASA_DB_PATH"`
}

func (cmd *incidentsListCmd) Run() error {
	if cmd.Status != "" {
		if err := validateStatus(cmd.Status); err != nil {
			return err
		}
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	items, err := store.ListIncidents(false)
	if err != nil {
		return fmt.Errorf("list incidents: %w", err)
	}

	if cmd.Status != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.Status == cmd.Status {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if cmd.JSON {
		out := make([]incidentJSON, len(items))
		for i, item := range items {
			out[i] = toIncidentJSON(item)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(items) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tSEVERITY\tLOCATION\tNOTICED")
	for _, item := range items {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Title,
			item.Status,
			item.Severity,
			item.Location,
			item.DateNoticed.Format(time.DateOnly),
		)
	}
	return w.Flush()
}

// --- add ---

type incidentsAddCmd struct {
	Title       string `help:"Incident title."                          required:""`
	Severity    string `help:"Severity (urgent|soon|whenever)."         required:"" enum:"urgent,soon,whenever"`
	Description string `help:"Description of the incident."`
	Location    string `help:"Location where the incident occurred."`
	ApplianceID *uint  `help:"Associated appliance ID."                 name:"appliance-id"`
	VendorID    *uint  `help:"Associated vendor ID."                    name:"vendor-id"`
	Notes       string `help:"Additional notes."`
	DBPath      string `help:"SQLite database path."                    env:"MICASA_DB_PATH"`
}

func (cmd *incidentsAddCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item := data.Incident{
		Title:       cmd.Title,
		Description: cmd.Description,
		Status:      data.IncidentStatusOpen,
		Severity:    cmd.Severity,
		DateNoticed: time.Now(),
		Location:    cmd.Location,
		ApplianceID: cmd.ApplianceID,
		VendorID:    cmd.VendorID,
		Notes:       cmd.Notes,
	}
	if err := store.CreateIncident(&item); err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	fmt.Println(item.ID)
	return nil
}

// --- show ---

type incidentsShowCmd struct {
	ID     string `arg:"" help:"Incident ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *incidentsShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetIncident(id)
	if err != nil {
		return fmt.Errorf("get incident %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toIncidentJSON(item))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", item.ID)
	fmt.Fprintf(w, "Title:\t%s\n", item.Title)
	fmt.Fprintf(w, "Status:\t%s\n", item.Status)
	fmt.Fprintf(w, "Severity:\t%s\n", item.Severity)
	if item.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", item.Description)
	}
	fmt.Fprintf(w, "Noticed:\t%s\n", item.DateNoticed.Format(time.DateOnly))
	if item.DateResolved != nil {
		fmt.Fprintf(w, "Resolved:\t%s\n", item.DateResolved.Format(time.DateOnly))
	}
	if item.Location != "" {
		fmt.Fprintf(w, "Location:\t%s\n", item.Location)
	}
	if item.CostCents != nil {
		fmt.Fprintf(w, "Cost:\t%s\n", data.FormatCents(*item.CostCents))
	}
	if item.Appliance.Name != "" {
		fmt.Fprintf(w, "Appliance:\t%s (ID %d)\n", item.Appliance.Name, item.Appliance.ID)
	}
	if item.Vendor.Name != "" {
		fmt.Fprintf(w, "Vendor:\t%s (ID %d)\n", item.Vendor.Name, item.Vendor.ID)
	}
	if item.Notes != "" {
		fmt.Fprintf(w, "Notes:\t%s\n", item.Notes)
	}
	return w.Flush()
}

// --- update ---

type incidentsUpdateCmd struct {
	ID          string  `arg:"" help:"Incident ID."`
	Title       *string `help:"New title."`
	Status      *string `help:"New status (open|in_progress|resolved)."`
	Severity    *string `help:"New severity (urgent|soon|whenever)."`
	Description *string `help:"New description."`
	Location    *string `help:"New location."`
	Notes       *string `help:"New notes."`
	ApplianceID *uint   `help:"New appliance ID (0 to clear)." name:"appliance-id"`
	VendorID    *uint   `help:"New vendor ID (0 to clear)."    name:"vendor-id"`
	DBPath      string  `help:"SQLite database path."          env:"MICASA_DB_PATH"`
}

func (cmd *incidentsUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	if cmd.Status != nil {
		if err := validateStatus(*cmd.Status); err != nil {
			return err
		}
	}
	if cmd.Severity != nil {
		if err := validateSeverity(*cmd.Severity); err != nil {
			return err
		}
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetIncident(id)
	if err != nil {
		return fmt.Errorf("get incident %d: %w", id, err)
	}

	if cmd.Title != nil {
		item.Title = *cmd.Title
	}
	if cmd.Status != nil {
		item.Status = *cmd.Status
	}
	if cmd.Severity != nil {
		item.Severity = *cmd.Severity
	}
	if cmd.Description != nil {
		item.Description = *cmd.Description
	}
	if cmd.Location != nil {
		item.Location = *cmd.Location
	}
	if cmd.Notes != nil {
		item.Notes = *cmd.Notes
	}
	if cmd.ApplianceID != nil {
		if *cmd.ApplianceID == 0 {
			item.ApplianceID = nil
		} else {
			item.ApplianceID = cmd.ApplianceID
		}
	}
	if cmd.VendorID != nil {
		if *cmd.VendorID == 0 {
			item.VendorID = nil
		} else {
			item.VendorID = cmd.VendorID
		}
	}

	if err := store.UpdateIncident(item); err != nil {
		return fmt.Errorf("update incident %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type incidentsDeleteCmd struct {
	ID     string `arg:"" help:"Incident ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *incidentsDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteIncident(id); err != nil {
		return fmt.Errorf("delete incident %d: %w", id, err)
	}
	return nil
}

// --- resolve ---

type incidentsResolveCmd struct {
	ID     string `arg:"" help:"Incident ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *incidentsResolveCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetIncident(id)
	if err != nil {
		return fmt.Errorf("get incident %d: %w", id, err)
	}

	now := time.Now()
	item.Status = data.IncidentStatusResolved
	item.DateResolved = &now

	if err := store.UpdateIncident(item); err != nil {
		return fmt.Errorf("resolve incident %d: %w", id, err)
	}
	return nil
}

func parseID(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID %q: must be a positive integer", s)
	}
	if n == 0 {
		return 0, fmt.Errorf("invalid ID: must be greater than zero")
	}
	return uint(n), nil
}
