package scanfile

import (
	"fmt"
	"os"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
	"github.com/eolseng/wow-markets/companion/internal/luasv"
)

const (
	DefaultVariableName = "WOW_MARKETS_DB"
	LegacyVariableName  = "WOW_MARKET_SCAN_DB"
)

func Load(path, variableName string) (exportfmt.Database, error) {
	if variableName != "" {
		return loadVariable(path, variableName)
	}

	database, primaryErr := loadVariable(path, DefaultVariableName)
	if primaryErr == nil {
		return database, nil
	}
	database, legacyErr := loadVariable(path, LegacyVariableName)
	if legacyErr == nil {
		return database, nil
	}
	return exportfmt.Database{}, fmt.Errorf(
		"load %s or legacy %s: %v; %v",
		DefaultVariableName,
		LegacyVariableName,
		primaryErr,
		legacyErr,
	)
}

func loadVariable(path, variableName string) (exportfmt.Database, error) {
	file, err := os.Open(path)
	if err != nil {
		return exportfmt.Database{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	root, err := luasv.ParseVariable(file, variableName)
	if err != nil {
		return exportfmt.Database{}, fmt.Errorf("parse SavedVariables: %w", err)
	}
	database, err := exportfmt.Decode(root)
	if err != nil {
		return exportfmt.Database{}, fmt.Errorf("decode export: %w", err)
	}
	return database, nil
}
