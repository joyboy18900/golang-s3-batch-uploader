package service

import (
	"strings"
	"testing"
)

func TestProcessCSV(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "trims whitespace from every field",
			input: "id, name \n 1 ,  foo  \n",
			want:  "id,name\n1,foo\n",
		},
		{
			name:    "malformed csv returns an error",
			input:   "id,name\n\"unterminated,foo\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processCSV(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("processCSV() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("processCSV() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("processCSV() = %q, want %q", got, tt.want)
			}
		})
	}
}
