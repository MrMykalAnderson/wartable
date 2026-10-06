package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MrMykalAnderson/wartable/internal/game"
)

// SaveFile is a saved game's on-disk JSON representation.
type SaveFile struct {
	ScenarioID string
	Turn       int
	TieBreak   game.TieBreak
	State      game.GameState
}

func loadSave(path string) (SaveFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SaveFile{}, fmt.Errorf("read %s: %w", path, err)
	}
	var save SaveFile
	if err := json.Unmarshal(raw, &save); err != nil {
		return SaveFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return save, nil
}

func writeSave(path string, save SaveFile) error {
	raw, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
