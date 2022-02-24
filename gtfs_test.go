package gtfs_test

import (
	"github.com/heimdalr/gtfs"
	"testing"
)

func TestGTFSDateTime_UnmarshalCSV(t *testing.T) {

	tests := []struct {
		name    string
		dt      int32
		csv     string
		wantErr bool
	}{
		{
			name:    "0:00:00",
			dt:      0,
			csv:     "0:00:00",
			wantErr: false,
		},
		{
			name:    "14:37:01",
			dt:      52621,
			csv:     "14:37:01",
			wantErr: false,
		},
		{
			name:    "a4:37:01",
			csv:     "a4:37:01",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dt gtfs.DateTime
			err := dt.UnmarshalCSV(tt.csv)
			if tt.wantErr {
				if err == nil {
					t.Errorf("UnmarshalCSV() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if dt.Int32 != tt.dt {
					t.Errorf("UnmarshalCSV() got %d, want %d", dt.Int32, tt.dt)
				}
			}
		})
	}
}

func TestGTFSDateTime_MarshalCSV(t *testing.T) {
	tests := []struct {
		name    string
		dt      int32
		csv     string
		wantErr bool
	}{
		{
			name:    "00:00:00",
			dt:      0,
			csv:     "00:00:00",
			wantErr: false,
		},
		{
			name:    "14:37:01",
			dt:      52621,
			csv:     "14:37:01",
			wantErr: false,
		},
		{
			name:    "11:29:00",
			dt:      41340,
			csv:     "11:29:00",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := gtfs.DateTime{Int32: tt.dt}
			csv, err := dt.MarshalCSV()
			if tt.wantErr {
				if err == nil {
					t.Errorf("MarshalCSV() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if csv != tt.csv {
					t.Errorf("MarshalCSV() got %s, want %s", csv, tt.csv)
				}
			}
		})
	}
}
