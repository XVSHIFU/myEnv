//go:build !windows

package core

func registeredInventory() ([]inventoryCandidate, []string) { return nil, nil }
