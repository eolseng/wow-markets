package scanfile

import (
	"fmt"
	"os"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
	"github.com/eolseng/wow-markets/companion/internal/luasv"
)

const DefaultVariableName = "WOW_MARKET_SCAN_DB"

func Load(path, variableName string) (exportfmt.Database, error) {
	file, err := os.Open(path)
	if err != nil {
		return exportfmt.Database{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	if variableName == "" {
		variableName = DefaultVariableName
	}

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
