package platforms

import (
	"errors"
)

// Load a slice of Destinations and Sources from values passed which the calling module shoukd have read from config
// If selectedDestinations is empty or nil, all destinations are loaded
func Load(sources interface{}, destinations interface{}, selectedSources []string, selectedDestinations []string) ([]Source, []Destination, error) {
	// Assert that destinations is a slice of interfaces
	configuredDestinations, ok := destinations.([]interface{})
	if !ok {
		return nil, nil, errors.New("assertion failed: destinations is not a slice of interfaces")
	}
	configuredSources, ok := sources.([]interface{})
	if !ok {
		return nil, nil, errors.New("assertion failed: sources is not a slice of interfaces")
	}

	// Create the slice of destinations and slice of sources that will be returned
	var destinationSlice []Destination
	var sourceSlice []Source
	// Iterate over the destinations
	for _, dest := range configuredDestinations {
		destMap, ok := dest.(map[string]interface{})
		if !ok {
			return nil, nil, errors.New("failed to convert destination to map")
		}
		destination, err := CreateDestination(destMap)
		if err != nil {
			return nil, nil, err
		}
		// If selectedDestinations is empty, add all configured destinations
		if len(selectedDestinations) == 0 {
			destinationSlice = append(destinationSlice, destination)
		} else {
			// Check if the destination name is in the selectedDestinations
			for _, selected := range selectedDestinations {
				if destination.GetName() == selected {
					destinationSlice = append(destinationSlice, destination)
					break
				}
			}
		}
	}
	// Iterate over the sources
	for _, src := range configuredSources {
		sourceMap, ok := src.(map[string]interface{})
		if !ok {
			return nil, nil, errors.New("failed to convert source to map")
		}
		source, err := CreateSource(sourceMap)
		if err != nil {
			return nil, nil, err
		}
		// If selectedSources is empty, add all configured sources
		if len(selectedSources) == 0 {
			sourceSlice = append(sourceSlice, source)
		} else {
			// Check if the source name is in the selectedSources
			for _, selected := range selectedSources {
				if source.GetName() == selected {
					sourceSlice = append(sourceSlice, source)
					break
				}
			}
		}
	}

	// Check if the selected sources and destinations are in the config
	if len(selectedSources) > 0 {
		for _, selected := range selectedSources {
			found := false
			for _, source := range sourceSlice {
				if source.GetName() == selected {
					found = true
					break
				}
			}
			if !found {
				return nil, nil, errors.New("selected source not found in config")
			}
		}
	}
	if len(selectedDestinations) > 0 {
		for _, selected := range selectedDestinations {
			found := false
			for _, destination := range destinationSlice {
				if destination.GetName() == selected {
					found = true
					break
				}
			}
			if !found {
				return nil, nil, errors.New("selected destination not found in config")
			}
		}
	}

	return sourceSlice, destinationSlice, nil

}
