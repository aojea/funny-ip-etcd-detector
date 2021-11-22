package main

import (
	"testing"
)

func Test_parseIPv4(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "valid IPv4",
			address: "1.1.1.1",
			want:    true,
		},
		{
			name:    "valid IPv4",
			address: "255.255.255.255",
			want:    true,
		},
		{
			name:    "valid IPv4",
			address: "0.0.0.0",
			want:    true,
		},
		{
			name:    "invalid IPv4",
			address: "0.00.0.0",
			want:    false,
		},
		{
			name:    "invalid IPv4",
			address: "10.001.20.30",
			want:    false,
		},
		{
			name:    "IPv6 (invalid)",
			address: "fd00:1:2:3::",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseIPv4(tt.address); got != tt.want {
				t.Errorf("parseIPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_iterateBucket(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "no invalid ips",
			dbPath:  "./testdata/snapshot_good.db",
			wantErr: false,
		},
		{
			name:    "invalid ips",
			dbPath:  "./testdata/snapshot_funny.db",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := iterateBucket(tt.dbPath, "key", 0, true); (err != nil) != tt.wantErr {
				t.Errorf("iterateBucket() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
