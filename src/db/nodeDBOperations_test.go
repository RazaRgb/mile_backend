package db

import "testing"

func TestPathPrefixes(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{name: "root", path: "0001", want: nil},
		{name: "single child", path: "0001.0001", want: []string{"0001"}},
		{name: "three levels", path: "0001.0001.0005", want: []string{"0001", "0001.0001"}},
		{name: "deep path", path: "0001.0001.0003.0001.0001", want: []string{"0001", "0001.0001", "0001.0001.0003", "0001.0001.0003.0001"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathPrefixes(tt.path)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
