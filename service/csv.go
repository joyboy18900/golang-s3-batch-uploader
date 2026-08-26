package service

import (
	"bytes"
	"encoding/csv"
	"io"
	"strings"
)

func processCSV(r io.Reader) ([]byte, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		for i, field := range record {
			record[i] = strings.TrimSpace(field)
		}
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.WriteAll(records); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
