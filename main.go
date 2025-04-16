package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agldw/postfix-bounce-parser/postfixutil"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		panic("No .env file found")
	}

	logDir := os.Getenv("LOG_DIR")

	if logDir == "" {
		panic("No LOG_DIR found")
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		panic("LOG_DIR not found")
	}
}

func main() {
	// Walk through all files in LOG_DIR including subdirectories
	err := filepath.Walk(os.Getenv("LOG_DIR"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Process each file
		files := []string{path}
		bounces := postfixutil.FindBounces(&files)
		
		// Skip if no bounces found
		if len(bounces) == 0 {
			return nil
		}
		
		// Create JSON output
		jsonData, err := json.Marshal(bounces)
		if err != nil {
			log.Printf("Failed to marshal JSON for %s: %v", path, err)
			return nil
		}
		
		// Create output filename with same structure but .json extension
		outputPath := path + ".json"
		outputExcelPath := path + ".xlsx"
		
		// Write to file
		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			log.Printf("Failed to write output file %s: %v", outputPath, err)
		} else {
			log.Printf("Processed %s -> %s (%d bounces)", path, outputPath, len(bounces))
		}

		// Create Excel file
		f := excelize.NewFile()
		sheetName := "Bounces"
		f.SetSheetName("Sheet1", sheetName)
		
		// Set headers
		headers := []string{"Date", "From", "To", "Relay", "Delay", "DSN", "Status", "Reason"}
		for i, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}
		
		// Set style for header row
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		})
		f.SetCellStyle(sheetName, "A1", string(rune('A'+len(headers)-1))+"1", style)
		
		// Add data
		for i, bounce := range bounces {
			row := i + 2 // Start from row 2 (after header)
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), bounce.Date.Format(time.RFC3339))
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), bounce.From)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), bounce.To)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), bounce.Relay)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), bounce.Delay)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), bounce.DSN)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), bounce.Status)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), bounce.Reason)
			// f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), bounce.IsHard())
		}
		
		// Save Excel file
		if err := f.SaveAs(outputExcelPath); err != nil {
			log.Printf("Failed to write Excel file %s: %v", outputExcelPath, err)
		} else {
			log.Printf("Created Excel file: %s", outputExcelPath)
		}
		
		return nil
	})
	
	if err != nil {
		log.Fatalf("Error walking through files: %v", err)
	}
}