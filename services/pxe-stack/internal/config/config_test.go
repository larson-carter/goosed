package config

import (
	"reflect"
	"testing"
)

func TestParsePortList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "dedupe and trim",
			input: "8081, 8082,8081,,8083",
			want:  []int{8081, 8082, 8083},
		},
		{
			name:    "invalid integer",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "out of range",
			input:   "70000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePortList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parsePortList() = %v, want %v", got, tt.want)
			}
		})
	}
}
