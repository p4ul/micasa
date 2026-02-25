// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cpcloud/micasa/internal/data"
)

type projectsCmd struct {
	List   projectsListCmd   `cmd:"" default:"withargs" help:"List projects."`
	Add    projectsAddCmd    `cmd:""                    help:"Add a new project."`
	Show   projectsShowCmd   `cmd:""                    help:"Show project details."`
	Update projectsUpdateCmd `cmd:""                    help:"Update a project."`
	Delete projectsDeleteCmd `cmd:""                    help:"Delete a project (soft-delete)."`
}

var validProjectStatuses = []string{
	data.ProjectStatusIdeating,
	data.ProjectStatusPlanned,
	data.ProjectStatusQuoted,
	data.ProjectStatusInProgress,
	data.ProjectStatusDelayed,
	data.ProjectStatusCompleted,
	data.ProjectStatusAbandoned,
}

func validateProjectStatus(s string) error {
	for _, v := range validProjectStatuses {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf(
		"invalid status %q (want %s)",
		s,
		strings.Join(validProjectStatuses, "|"),
	)
}

type projectJSON struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	ProjectType string `json:"project_type"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	BudgetCents *int64 `json:"budget_cents,omitempty"`
	ActualCents *int64 `json:"actual_cents,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toProjectJSON(item data.Project) projectJSON {
	j := projectJSON{
		ID:          item.ID,
		Title:       item.Title,
		ProjectType: item.ProjectType.Name,
		Status:      item.Status,
		Description: item.Description,
		BudgetCents: item.BudgetCents,
		ActualCents: item.ActualCents,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
	if item.StartDate != nil {
		j.StartDate = item.StartDate.Format(time.DateOnly)
	}
	if item.EndDate != nil {
		j.EndDate = item.EndDate.Format(time.DateOnly)
	}
	return j
}

// --- list ---

type projectsListCmd struct {
	JSON   bool   `help:"Output as JSON."           short:"j"`
	Status string `help:"Filter by project status." short:"s"`
	DBPath string `help:"SQLite database path."     env:"MICASA_DB_PATH"`
}

func (cmd *projectsListCmd) Run() error {
	if cmd.Status != "" {
		if err := validateProjectStatus(cmd.Status); err != nil {
			return err
		}
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	items, err := store.ListProjects(false)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
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
		out := make([]projectJSON, len(items))
		for i, item := range items {
			out[i] = toProjectJSON(item)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(items) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tTYPE\tSTATUS\tBUDGET")
	for _, item := range items {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Title,
			item.ProjectType.Name,
			item.Status,
			data.FormatOptionalCents(item.BudgetCents),
		)
	}
	return w.Flush()
}

// --- add ---

type projectsAddCmd struct {
	Title       string `help:"Project title."                       required:""`
	Type        string `help:"Project type (e.g. Flooring, HVAC)."  required:"" name:"type"`
	Status      string `help:"Initial status."                      default:"ideating"`
	Description string `help:"Project description."`
	Budget      string `help:"Budget amount (e.g. 1234.56)."`
	DBPath      string `help:"SQLite database path."                env:"MICASA_DB_PATH"`
}

func (cmd *projectsAddCmd) Run() error {
	if err := validateProjectStatus(cmd.Status); err != nil {
		return err
	}

	var budgetCents *int64
	if cmd.Budget != "" {
		cents, err := data.ParseRequiredCents(cmd.Budget)
		if err != nil {
			return fmt.Errorf("invalid budget: %w", err)
		}
		budgetCents = &cents
	}

	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	typeID, err := resolveProjectType(store, cmd.Type)
	if err != nil {
		return err
	}

	item := data.Project{
		Title:         cmd.Title,
		ProjectTypeID: typeID,
		Status:        cmd.Status,
		Description:   cmd.Description,
		BudgetCents:   budgetCents,
	}
	if err := store.CreateProject(&item); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	fmt.Println(item.ID)
	return nil
}

func resolveProjectType(store *data.Store, name string) (uint, error) {
	types, err := store.ProjectTypes()
	if err != nil {
		return 0, fmt.Errorf("list project types: %w", err)
	}
	for _, pt := range types {
		if strings.EqualFold(pt.Name, name) {
			return pt.ID, nil
		}
	}
	names := make([]string, len(types))
	for i, pt := range types {
		names[i] = pt.Name
	}
	return 0, fmt.Errorf(
		"unknown project type %q (want %s)",
		name,
		strings.Join(names, "|"),
	)
}

// --- show ---

type projectsShowCmd struct {
	ID     string `arg:"" help:"Project ID."`
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *projectsShowCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetProject(id)
	if err != nil {
		return fmt.Errorf("get project %d: %w", id, err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toProjectJSON(item))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", item.ID)
	fmt.Fprintf(w, "Title:\t%s\n", item.Title)
	fmt.Fprintf(w, "Type:\t%s\n", item.ProjectType.Name)
	fmt.Fprintf(w, "Status:\t%s\n", item.Status)
	if item.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", item.Description)
	}
	if item.StartDate != nil {
		fmt.Fprintf(w, "Start:\t%s\n", item.StartDate.Format(time.DateOnly))
	}
	if item.EndDate != nil {
		fmt.Fprintf(w, "End:\t%s\n", item.EndDate.Format(time.DateOnly))
	}
	if item.BudgetCents != nil {
		fmt.Fprintf(w, "Budget:\t%s\n", data.FormatCents(*item.BudgetCents))
	}
	if item.ActualCents != nil {
		fmt.Fprintf(w, "Actual:\t%s\n", data.FormatCents(*item.ActualCents))
	}
	return w.Flush()
}

// --- update ---

type projectsUpdateCmd struct {
	ID          string  `arg:"" help:"Project ID."`
	Title       *string `help:"New title."`
	Status      *string `help:"New status."`
	Type        *string `help:"New project type."  name:"type"`
	Description *string `help:"New description."`
	Budget      *string `help:"New budget (e.g. 1234.56, empty to clear)."`
	DBPath      string  `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *projectsUpdateCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	if cmd.Status != nil {
		if err := validateProjectStatus(*cmd.Status); err != nil {
			return err
		}
	}

	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	item, err := store.GetProject(id)
	if err != nil {
		return fmt.Errorf("get project %d: %w", id, err)
	}

	if cmd.Title != nil {
		item.Title = *cmd.Title
	}
	if cmd.Status != nil {
		item.Status = *cmd.Status
	}
	if cmd.Description != nil {
		item.Description = *cmd.Description
	}
	if cmd.Type != nil {
		typeID, err := resolveProjectType(store, *cmd.Type)
		if err != nil {
			return err
		}
		item.ProjectTypeID = typeID
	}
	if cmd.Budget != nil {
		if *cmd.Budget == "" {
			item.BudgetCents = nil
		} else {
			cents, err := data.ParseRequiredCents(*cmd.Budget)
			if err != nil {
				return fmt.Errorf("invalid budget: %w", err)
			}
			item.BudgetCents = &cents
		}
	}

	if err := store.UpdateProject(item); err != nil {
		return fmt.Errorf("update project %d: %w", id, err)
	}
	return nil
}

// --- delete ---

type projectsDeleteCmd struct {
	ID     string `arg:"" help:"Project ID."`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

func (cmd *projectsDeleteCmd) Run() error {
	id, err := parseID(cmd.ID)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteProject(id); err != nil {
		return fmt.Errorf("delete project %d: %w", id, err)
	}
	return nil
}
