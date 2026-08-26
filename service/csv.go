package service

import (
	"encoding/csv"
	"io"
)

func validateCSV(r io.Reader) error {
	_, err := csv.NewReader(r).ReadAll()
	return err
}
