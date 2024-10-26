/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"regexp"
	"testing"
)

func TestGetRandomObject(t *testing.T) {
	// Test case: input is an empty slice
	t.Run("Empty slice returns empty string", func(t *testing.T) {
		input := []string{}
		if result := GetRandomObject(input); result != "" {
			t.Errorf("Expected empty string, got %v", result)
		}
	})

	// Test case: input has one element
	t.Run("Single element slice returns the element itself", func(t *testing.T) {
		input := []string{"onlyElement"}
		if result := GetRandomObject(input); result != "onlyElement" {
			t.Errorf("Expected 'onlyElement', got %v", result)
		}
	})

	// Test case: input has multiple elements
	t.Run("Multiple element slice returns a random element", func(t *testing.T) {
		input := []string{"one", "two", "three"}
		// Using a set to confirm returned values are within input values
		results := make(map[string]bool)
		for i := 0; i < 100; i++ {
			result := GetRandomObject(input)
			if !contains(input, result) {
				t.Errorf("Unexpected result: %v", result)
			}
			results[result] = true
		}

		// Assert that randomness covers at least two elements
		if len(results) < 2 {
			t.Errorf("Expected random distribution, but got only %d unique results", len(results))
		}
	})
}

func TestGenerateUUID(t *testing.T) {
	// Test case: Ensure that each generated UUID is unique
	t.Run("Generated UUIDs should be unique", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			id := GenerateUUID()
			if ids[id] {
				t.Errorf("Duplicate UUID found: %v", id)
			}
			ids[id] = true
		}
	})

	// Test case: Validate that each generated UUID matches the correct UUID format
	t.Run("Generated UUID should follow valid UUID format", func(t *testing.T) {
		id := GenerateUUID()
		uuidFormat := `^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`
		match, _ := regexp.MatchString(uuidFormat, id)
		if !match {
			t.Errorf("Invalid UUID format: %v", id)
		}
	})
}

func TestContains(t *testing.T) {
	// Test case: Empty slice should return false
	t.Run("Empty slice should return false", func(t *testing.T) {
		slice := []string{}
		value := "test"
		if result := contains(slice, value); result {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// Test case: Slice contains the target value
	t.Run("Slice contains the value", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		value := "banana"
		if result := contains(slice, value); !result {
			t.Errorf("Expected true, got %v", result)
		}
	})

	// Test case: Slice does not contain the target value
	t.Run("Slice does not contain the value", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		value := "orange"
		if result := contains(slice, value); result {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// Test case: Slice contains multiple instances of the target value
	t.Run("Slice with multiple instances of the value", func(t *testing.T) {
		slice := []string{"apple", "banana", "banana", "cherry"}
		value := "banana"
		if result := contains(slice, value); !result {
			t.Errorf("Expected true, got %v", result)
		}
	})
}
