package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
)

type Units struct {
	SolarRadiation string // kWh/day
	CoolingLoad    string // kWh/day
	Electricity    string // kWh/day
	Cost           string // $/kWh
	Savings        string // $/day
}

type Assumptions struct {
	Units              Units
	Location           string
	BuildingType       string
	AC_COP             float64
	SHGC               float64
	WWR                float64
	TransmissionFactor float64
	TimeLagFactor      float64
	MedicalEquipFactor float64
	ElectricityCost    float64
}

type Result struct {
	Assumptions         Assumptions
	TotalSolarReduction float64
	CoolingLoadReduced  float64
	ElectricitySaved    float64
	AnnualCostSaved     float64
}

type ResultOutput struct {
	// metadata
	Timestamp    string `json:"timestamp"`
	Location     string `json:"location"`
	BuildingType string `json:"building_type"`

	// inputs
	SolarReduction     float64 `json:"solar_reduction_kwh_day"`
	ElectricityCost    float64 `json:"electricity_cost_per_kwh"`
	AC_COP             float64 `json:"ac_cop"`
	SHGC               float64 `json:"shgc"`
	WWR                float64 `json:"wwr"`
	TransmissionFactor float64 `json:"transmission_factor"`
	TimeLagFactor      float64 `json:"time_lag_factor"`
	MedicalEquipFactor float64 `json:"medical_equip_factor"`

	// results
	CoolingLoadReduced float64 `json:"cooling_load_reduced_kwh_day"`
	ElectricitySaved   float64 `json:"electricity_saved_kwh_day"`
	DailyCostSaved     float64 `json:"daily_cost_saved_usd"`
}

type Config struct {
	Location           string
	OutputDir          string
	SolarReduction     float64
	ElectricityCost    float64
	AC_COP             float64
	SHGC               float64
	WWR                float64
	TransmissionFactor float64
	TimeLagFactor      float64
	MedicalEquipFactor float64
}

func DefaultConfig() Config {
	return Config{
		Location:           "Sacramento",
		AC_COP:             4.0,  // ASHRAE 90.1-2019
		SHGC:               0.25, // CA Title 24 2022
		WWR:                0.40, // DOE Reference Building
		TransmissionFactor: 0.80,
		TimeLagFactor:      0.95,
		MedicalEquipFactor: 1.15,
		OutputDir:          "results",
	}
}

func calculateCoolingSavings(config Config) Result {
	coolingLoadReduced := config.SolarReduction *
		config.SHGC *
		config.TransmissionFactor *
		config.TimeLagFactor *
		config.MedicalEquipFactor

	electricitySaved := coolingLoadReduced / config.AC_COP
	annualCostSaved := electricitySaved * config.ElectricityCost * 365

	return Result{
		TotalSolarReduction: config.SolarReduction,
		CoolingLoadReduced:  coolingLoadReduced,
		ElectricitySaved:    electricitySaved,
		AnnualCostSaved:     annualCostSaved,
		Assumptions: Assumptions{
			Location:           config.Location,
			BuildingType:       "Medical Clinic",
			AC_COP:             config.AC_COP,
			SHGC:               config.SHGC,
			WWR:                config.WWR,
			TransmissionFactor: config.TransmissionFactor,
			TimeLagFactor:      config.TimeLagFactor,
			MedicalEquipFactor: config.MedicalEquipFactor,
			ElectricityCost:    config.ElectricityCost,
			Units: Units{
				SolarRadiation: "kWh/day",
				CoolingLoad:    "kWh/day",
				Electricity:    "kWh/day",
				Cost:           "$/kWh",
				Savings:        "$/year",
			},
		},
	}
}

func saveResults(result Result, config Config) error {
	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")

	output := ResultOutput{
		Timestamp:          time.Now().Format(time.RFC3339),
		Location:           result.Assumptions.Location,
		BuildingType:       result.Assumptions.BuildingType,
		SolarReduction:     result.TotalSolarReduction,
		ElectricityCost:    result.Assumptions.ElectricityCost,
		AC_COP:             result.Assumptions.AC_COP,
		SHGC:               result.Assumptions.SHGC,
		WWR:                result.Assumptions.WWR,
		TransmissionFactor: result.Assumptions.TransmissionFactor,
		TimeLagFactor:      result.Assumptions.TimeLagFactor,
		MedicalEquipFactor: result.Assumptions.MedicalEquipFactor,
		CoolingLoadReduced: result.CoolingLoadReduced,
		ElectricitySaved:   result.ElectricitySaved,
		DailyCostSaved:     result.AnnualCostSaved,
	}

	jsonPath := filepath.Join(config.OutputDir, fmt.Sprintf("solar_cooling_%s.json", timestamp))
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	csvPath := filepath.Join(config.OutputDir, fmt.Sprintf("solar_cooling_%s.csv", timestamp))
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	headers := []string{
		"Timestamp", "Location", "Building Type",
		"Solar Reduction (kWh/day)", "Electricity Cost ($/kWh)",
		"AC COP", "SHGC", "WWR",
		"Transmission Factor", "Time Lag Factor", "Medical Equipment Factor",
		"Cooling Load Reduced (kWh/day)", "Electricity Saved (kWh/day)",
		"Daily Cost Saved ($)",
	}

	data := []string{
		output.Timestamp, output.Location, output.BuildingType,
		fmt.Sprintf("%.2f", output.SolarReduction),
		fmt.Sprintf("%.3f", output.ElectricityCost),
		fmt.Sprintf("%.1f", output.AC_COP),
		fmt.Sprintf("%.2f", output.SHGC),
		fmt.Sprintf("%.2f", output.WWR),
		fmt.Sprintf("%.2f", output.TransmissionFactor),
		fmt.Sprintf("%.2f", output.TimeLagFactor),
		fmt.Sprintf("%.2f", output.MedicalEquipFactor),
		fmt.Sprintf("%.2f", output.CoolingLoadReduced),
		fmt.Sprintf("%.2f", output.ElectricitySaved),
		fmt.Sprintf("%.2f", output.DailyCostSaved),
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %v", err)
	}
	if err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write CSV data: %v", err)
	}

	return nil
}

func main() {
	config := DefaultConfig()

	var (
		verbose     bool
		showVersion bool
	)

	const version = "1.4.0"

	pflag.Float64VarP(&config.SolarReduction, "reduction", "r", 0.0,
		"Total solar radiation reduction in kWh/day")
	pflag.Float64VarP(&config.ElectricityCost, "cost", "c", 0.0,
		"Electricity cost in $/kWh")

	pflag.StringVarP(&config.Location, "location", "l", config.Location,
		"Building location")
	pflag.Float64Var(&config.AC_COP, "cop", config.AC_COP,
		"Air conditioning Coefficient of Performance")
	pflag.Float64Var(&config.SHGC, "shgc", config.SHGC,
		"Solar Heat Gain Coefficient")
	pflag.Float64Var(&config.WWR, "wwr", config.WWR,
		"Window to Wall Ratio")
	pflag.StringVarP(&config.OutputDir, "output", "o", config.OutputDir,
		"Output directory for CSV and JSON files")

	pflag.BoolVarP(&verbose, "verbose", "v", false,
		"Show detailed assumptions and calculations")
	pflag.BoolVarP(&showVersion, "version", "V", false,
		"Show program version")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Solar Cooling Energy Calculator for Medical Clinics v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  calculator [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Required Flags:\n")
		fmt.Fprintf(os.Stderr, "  -r, --reduction float   Total solar radiation reduction in kWh/day\n")
		fmt.Fprintf(os.Stderr, "  -c, --cost float        Electricity cost in $/kWh\n\n")
		fmt.Fprintf(os.Stderr, "Optional Flags (with defaults):\n")
		fmt.Fprintf(os.Stderr, "      --cop float         AC Coefficient of Performance (default: %.1f)\n", config.AC_COP)
		fmt.Fprintf(os.Stderr, "      --shgc float        Solar Heat Gain Coefficient (default: %.2f)\n", config.SHGC)
		fmt.Fprintf(os.Stderr, "      --wwr float         Window to Wall Ratio (default: %.2f)\n", config.WWR)
		fmt.Fprintf(os.Stderr, "  -l, --location string   Building location (default: %s)\n", config.Location)
		fmt.Fprintf(os.Stderr, "  -o, --output string     Output directory (default: %s)\n\n", config.OutputDir)
		fmt.Fprintf(os.Stderr, "Other Options:\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose          Show detailed assumptions and calculations\n")
		fmt.Fprintf(os.Stderr, "  -V, --version          Show program version\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  calculator -r 100 -c 0.15\n")
		fmt.Fprintf(os.Stderr, "  calculator --reduction 150.5 --cost 0.12 --cop 3.5 --shgc 0.3 -o results\n")
	}

	pflag.Parse()

	if showVersion {
		fmt.Printf("Solar Cooling Energy Calculator v%s\n", version)
		os.Exit(0)
	}

	if config.SolarReduction <= 0 {
		fmt.Println("Error: Solar reduction must be a positive number")
		pflag.Usage()
		os.Exit(1)
	}

	if config.ElectricityCost <= 0 {
		fmt.Println("Error: Electricity cost must be a positive number")
		pflag.Usage()
		os.Exit(1)
	}

	if config.SHGC <= 0 || config.SHGC > 1 {
		fmt.Println("Error: SHGC must be between 0 and 1")
		os.Exit(1)
	}

	if config.WWR <= 0 || config.WWR > 1 {
		fmt.Println("Error: WWR must be between 0 and 1")
		os.Exit(1)
	}

	if config.AC_COP <= 0 {
		fmt.Println("Error: COP must be positive")
		os.Exit(1)
	}

	result := calculateCoolingSavings(config)

	if err := saveResults(result, config); err != nil {
		fmt.Printf("Error saving results: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nCalculation Results (Daily):\n")
	fmt.Printf("Location: %s\n", result.Assumptions.Location)
	fmt.Printf("Building type: %s\n", result.Assumptions.BuildingType)

	fmt.Printf("\nInputs:\n")
	fmt.Printf("Total solar radiation reduction: %.2f %s\n",
		result.TotalSolarReduction,
		result.Assumptions.Units.SolarRadiation)
	fmt.Printf("Electricity cost: %.3f %s\n",
		result.Assumptions.ElectricityCost,
		result.Assumptions.Units.Cost)

	if verbose {
		fmt.Printf("AC COP: %.1f\n", result.Assumptions.AC_COP)
		fmt.Printf("Solar Heat Gain Coefficient: %.2f\n", result.Assumptions.SHGC)
		fmt.Printf("Window-to-Wall Ratio: %.2f\n", result.Assumptions.WWR)
	}

	fmt.Printf("\nResults:\n")
	fmt.Printf("Total cooling load reduced: %.2f %s\n",
		result.CoolingLoadReduced,
		result.Assumptions.Units.CoolingLoad)
	fmt.Printf("Total electricity saved: %.2f %s\n",
		result.ElectricitySaved,
		result.Assumptions.Units.Electricity)
	fmt.Printf("Annual cost savings: %.2f %s\n",
		result.AnnualCostSaved,
		result.Assumptions.Units.Savings)

	if verbose {
		fmt.Printf("\nDetailed Assumptions:\n")
		fmt.Printf("Transmission Factor: %.2f\n", result.Assumptions.TransmissionFactor)
		fmt.Printf("Time Lag Factor: %.2f\n", result.Assumptions.TimeLagFactor)
		fmt.Printf("Medical Equipment Factor: %.2f\n", result.Assumptions.MedicalEquipFactor)
	}
}
