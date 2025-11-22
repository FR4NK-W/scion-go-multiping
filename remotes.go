package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type SCIONDestination struct {
	Address      string `json:"address"`
	IpAddress    string `json:"ip_address"`
	Name         string `json:"name"`
	ScionVersion string `json:"scion_version"`
}

type IPDestination struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Destinations struct {
	SCIONDestinations []SCIONDestination `json:"scion_destinations"`
	IPDestinations    []IPDestination    `json:"ip_destinations"`
}

func parseRemotesJSON(filename string) (*Destinations, error) {
	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the file contents
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the JSON data into the Destinations struct
	var destinations Destinations
	if err := json.Unmarshal(data, &destinations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &destinations, nil
}
