// Copyright 2026 Phillip Cloud
// Licensed under the Apache License, Version 2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cpcloud/micasa/internal/data"
	"gorm.io/gorm"
)

type houseCmd struct {
	Show   houseShowCmd   `cmd:"" default:"withargs" help:"Show house profile."`
	Update houseUpdateCmd `cmd:""                    help:"Update house profile."`
}

// --- show ---

type houseShowCmd struct {
	JSON   bool   `help:"Output as JSON." short:"j"`
	DBPath string `help:"SQLite database path." env:"MICASA_DB_PATH"`
}

type houseProfileJSON struct {
	ID               uint    `json:"id"`
	Nickname         string  `json:"nickname,omitempty"`
	AddressLine1     string  `json:"address_line_1,omitempty"`
	AddressLine2     string  `json:"address_line_2,omitempty"`
	City             string  `json:"city,omitempty"`
	State            string  `json:"state,omitempty"`
	PostalCode       string  `json:"postal_code,omitempty"`
	YearBuilt        int     `json:"year_built,omitempty"`
	SquareFeet       int     `json:"square_feet,omitempty"`
	LotSquareFeet    int     `json:"lot_square_feet,omitempty"`
	Bedrooms         int     `json:"bedrooms,omitempty"`
	Bathrooms        float64 `json:"bathrooms,omitempty"`
	FoundationType   string  `json:"foundation_type,omitempty"`
	WiringType       string  `json:"wiring_type,omitempty"`
	RoofType         string  `json:"roof_type,omitempty"`
	ExteriorType     string  `json:"exterior_type,omitempty"`
	HeatingType      string  `json:"heating_type,omitempty"`
	CoolingType      string  `json:"cooling_type,omitempty"`
	WaterSource      string  `json:"water_source,omitempty"`
	SewerType        string  `json:"sewer_type,omitempty"`
	ParkingType      string  `json:"parking_type,omitempty"`
	BasementType     string  `json:"basement_type,omitempty"`
	InsuranceCarrier string  `json:"insurance_carrier,omitempty"`
	InsurancePolicy  string  `json:"insurance_policy,omitempty"`
	InsuranceRenewal *string `json:"insurance_renewal,omitempty"`
	PropertyTaxCents *int64  `json:"property_tax_cents,omitempty"`
	HOAName          string  `json:"hoa_name,omitempty"`
	HOAFeeCents      *int64  `json:"hoa_fee_cents,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func toHouseProfileJSON(p data.HouseProfile) houseProfileJSON {
	j := houseProfileJSON{
		ID:               p.ID,
		Nickname:         p.Nickname,
		AddressLine1:     p.AddressLine1,
		AddressLine2:     p.AddressLine2,
		City:             p.City,
		State:            p.State,
		PostalCode:       p.PostalCode,
		YearBuilt:        p.YearBuilt,
		SquareFeet:       p.SquareFeet,
		LotSquareFeet:    p.LotSquareFeet,
		Bedrooms:         p.Bedrooms,
		Bathrooms:        p.Bathrooms,
		FoundationType:   p.FoundationType,
		WiringType:       p.WiringType,
		RoofType:         p.RoofType,
		ExteriorType:     p.ExteriorType,
		HeatingType:      p.HeatingType,
		CoolingType:      p.CoolingType,
		WaterSource:      p.WaterSource,
		SewerType:        p.SewerType,
		ParkingType:      p.ParkingType,
		BasementType:     p.BasementType,
		InsuranceCarrier: p.InsuranceCarrier,
		InsurancePolicy:  p.InsurancePolicy,
		PropertyTaxCents: p.PropertyTaxCents,
		HOAName:          p.HOAName,
		HOAFeeCents:      p.HOAFeeCents,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
	}
	if p.InsuranceRenewal != nil {
		s := p.InsuranceRenewal.Format(time.DateOnly)
		j.InsuranceRenewal = &s
	}
	return j
}

func (cmd *houseShowCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	profile, err := store.HouseProfile()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no house profile configured -- run 'micasa house update' to create one")
		}
		return fmt.Errorf("get house profile: %w", err)
	}

	if cmd.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(toHouseProfileJSON(profile))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if profile.Nickname != "" {
		fmt.Fprintf(w, "Nickname:\t%s\n", profile.Nickname)
	}
	if profile.AddressLine1 != "" {
		fmt.Fprintf(w, "Address:\t%s\n", profile.AddressLine1)
	}
	if profile.AddressLine2 != "" {
		fmt.Fprintf(w, "Address 2:\t%s\n", profile.AddressLine2)
	}
	if profile.City != "" {
		fmt.Fprintf(w, "City:\t%s\n", profile.City)
	}
	if profile.State != "" {
		fmt.Fprintf(w, "State:\t%s\n", profile.State)
	}
	if profile.PostalCode != "" {
		fmt.Fprintf(w, "Postal code:\t%s\n", profile.PostalCode)
	}
	if profile.YearBuilt != 0 {
		fmt.Fprintf(w, "Year built:\t%d\n", profile.YearBuilt)
	}
	if profile.SquareFeet != 0 {
		fmt.Fprintf(w, "Sq ft:\t%d\n", profile.SquareFeet)
	}
	if profile.LotSquareFeet != 0 {
		fmt.Fprintf(w, "Lot sq ft:\t%d\n", profile.LotSquareFeet)
	}
	if profile.Bedrooms != 0 {
		fmt.Fprintf(w, "Bedrooms:\t%d\n", profile.Bedrooms)
	}
	if profile.Bathrooms != 0 {
		fmt.Fprintf(w, "Bathrooms:\t%g\n", profile.Bathrooms)
	}
	if profile.FoundationType != "" {
		fmt.Fprintf(w, "Foundation:\t%s\n", profile.FoundationType)
	}
	if profile.WiringType != "" {
		fmt.Fprintf(w, "Wiring:\t%s\n", profile.WiringType)
	}
	if profile.RoofType != "" {
		fmt.Fprintf(w, "Roof:\t%s\n", profile.RoofType)
	}
	if profile.ExteriorType != "" {
		fmt.Fprintf(w, "Exterior:\t%s\n", profile.ExteriorType)
	}
	if profile.HeatingType != "" {
		fmt.Fprintf(w, "Heating:\t%s\n", profile.HeatingType)
	}
	if profile.CoolingType != "" {
		fmt.Fprintf(w, "Cooling:\t%s\n", profile.CoolingType)
	}
	if profile.WaterSource != "" {
		fmt.Fprintf(w, "Water:\t%s\n", profile.WaterSource)
	}
	if profile.SewerType != "" {
		fmt.Fprintf(w, "Sewer:\t%s\n", profile.SewerType)
	}
	if profile.ParkingType != "" {
		fmt.Fprintf(w, "Parking:\t%s\n", profile.ParkingType)
	}
	if profile.BasementType != "" {
		fmt.Fprintf(w, "Basement:\t%s\n", profile.BasementType)
	}
	if profile.InsuranceCarrier != "" {
		fmt.Fprintf(w, "Insurance:\t%s\n", profile.InsuranceCarrier)
	}
	if profile.InsurancePolicy != "" {
		fmt.Fprintf(w, "Policy:\t%s\n", profile.InsurancePolicy)
	}
	if profile.InsuranceRenewal != nil {
		fmt.Fprintf(w, "Renewal:\t%s\n", profile.InsuranceRenewal.Format(time.DateOnly))
	}
	if profile.PropertyTaxCents != nil {
		fmt.Fprintf(w, "Property tax:\t%s\n", data.FormatCents(*profile.PropertyTaxCents))
	}
	if profile.HOAName != "" {
		fmt.Fprintf(w, "HOA:\t%s\n", profile.HOAName)
	}
	if profile.HOAFeeCents != nil {
		fmt.Fprintf(w, "HOA fee:\t%s\n", data.FormatCents(*profile.HOAFeeCents))
	}
	return w.Flush()
}

// --- update ---

type houseUpdateCmd struct {
	Nickname         *string  `help:"House nickname."`
	Address          *string  `help:"Street address (line 1)."`
	AddressLine2     *string  `help:"Address line 2."                      name:"address-line-2"`
	City             *string  `help:"City."`
	State            *string  `help:"State/province."`
	PostalCode       *string  `help:"Postal/ZIP code."                     name:"postal-code"`
	YearBuilt        *int     `help:"Year built."                          name:"year-built"`
	Sqft             *int     `help:"Living area (sq ft)."`
	LotSqft          *int     `help:"Lot size (sq ft)."                    name:"lot-sqft"`
	Bedrooms         *int     `help:"Number of bedrooms."`
	Bathrooms        *float64 `help:"Number of bathrooms."`
	FoundationType   *string  `help:"Foundation type."                     name:"foundation-type"`
	WiringType       *string  `help:"Wiring type."                         name:"wiring-type"`
	RoofType         *string  `help:"Roof type."                           name:"roof-type"`
	ExteriorType     *string  `help:"Exterior type."                       name:"exterior-type"`
	HeatingType      *string  `help:"Heating type."                        name:"heating-type"`
	CoolingType      *string  `help:"Cooling type."                        name:"cooling-type"`
	WaterSource      *string  `help:"Water source."                        name:"water-source"`
	SewerType        *string  `help:"Sewer type."                          name:"sewer-type"`
	ParkingType      *string  `help:"Parking type."                        name:"parking-type"`
	BasementType     *string  `help:"Basement type."                       name:"basement-type"`
	InsuranceCarrier *string  `help:"Insurance carrier."                   name:"insurance-carrier"`
	InsurancePolicy  *string  `help:"Insurance policy number."             name:"insurance-policy"`
	InsuranceRenewal *string  `help:"Insurance renewal date (YYYY-MM-DD)." name:"insurance-renewal"`
	PropertyTaxCents *int64   `help:"Annual property tax (cents)."         name:"property-tax-cents"`
	HOAName          *string  `help:"HOA name."                            name:"hoa-name"`
	HOAFeeCents      *int64   `help:"Monthly HOA fee (cents)."             name:"hoa-fee-cents"`
	DBPath           string   `help:"SQLite database path."                env:"MICASA_DB_PATH"`
}

func (cmd *houseUpdateCmd) Run() error {
	store, err := openStore(cmd.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	var renewalDate *time.Time
	if cmd.InsuranceRenewal != nil {
		t, err := time.Parse(time.DateOnly, *cmd.InsuranceRenewal)
		if err != nil {
			return fmt.Errorf(
				"invalid insurance renewal date %q: expected YYYY-MM-DD",
				*cmd.InsuranceRenewal,
			)
		}
		renewalDate = &t
	}

	profile, err := store.HouseProfile()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = data.HouseProfile{}
		cmd.applyFlags(&profile, renewalDate)
		if err := store.CreateHouseProfile(profile); err != nil {
			return fmt.Errorf("create house profile: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get house profile: %w", err)
	}

	cmd.applyFlags(&profile, renewalDate)
	if err := store.UpdateHouseProfile(profile); err != nil {
		return fmt.Errorf("update house profile: %w", err)
	}
	return nil
}

func (cmd *houseUpdateCmd) applyFlags(p *data.HouseProfile, renewalDate *time.Time) {
	if cmd.Nickname != nil {
		p.Nickname = *cmd.Nickname
	}
	if cmd.Address != nil {
		p.AddressLine1 = *cmd.Address
	}
	if cmd.AddressLine2 != nil {
		p.AddressLine2 = *cmd.AddressLine2
	}
	if cmd.City != nil {
		p.City = *cmd.City
	}
	if cmd.State != nil {
		p.State = *cmd.State
	}
	if cmd.PostalCode != nil {
		p.PostalCode = *cmd.PostalCode
	}
	if cmd.YearBuilt != nil {
		p.YearBuilt = *cmd.YearBuilt
	}
	if cmd.Sqft != nil {
		p.SquareFeet = *cmd.Sqft
	}
	if cmd.LotSqft != nil {
		p.LotSquareFeet = *cmd.LotSqft
	}
	if cmd.Bedrooms != nil {
		p.Bedrooms = *cmd.Bedrooms
	}
	if cmd.Bathrooms != nil {
		p.Bathrooms = *cmd.Bathrooms
	}
	if cmd.FoundationType != nil {
		p.FoundationType = *cmd.FoundationType
	}
	if cmd.WiringType != nil {
		p.WiringType = *cmd.WiringType
	}
	if cmd.RoofType != nil {
		p.RoofType = *cmd.RoofType
	}
	if cmd.ExteriorType != nil {
		p.ExteriorType = *cmd.ExteriorType
	}
	if cmd.HeatingType != nil {
		p.HeatingType = *cmd.HeatingType
	}
	if cmd.CoolingType != nil {
		p.CoolingType = *cmd.CoolingType
	}
	if cmd.WaterSource != nil {
		p.WaterSource = *cmd.WaterSource
	}
	if cmd.SewerType != nil {
		p.SewerType = *cmd.SewerType
	}
	if cmd.ParkingType != nil {
		p.ParkingType = *cmd.ParkingType
	}
	if cmd.BasementType != nil {
		p.BasementType = *cmd.BasementType
	}
	if cmd.InsuranceCarrier != nil {
		p.InsuranceCarrier = *cmd.InsuranceCarrier
	}
	if cmd.InsurancePolicy != nil {
		p.InsurancePolicy = *cmd.InsurancePolicy
	}
	if renewalDate != nil {
		p.InsuranceRenewal = renewalDate
	}
	if cmd.PropertyTaxCents != nil {
		p.PropertyTaxCents = cmd.PropertyTaxCents
	}
	if cmd.HOAName != nil {
		p.HOAName = *cmd.HOAName
	}
	if cmd.HOAFeeCents != nil {
		p.HOAFeeCents = cmd.HOAFeeCents
	}
}
