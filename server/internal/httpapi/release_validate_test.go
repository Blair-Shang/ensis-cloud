package httpapi

import "testing"

func TestValidateDownloadURL(t *testing.T) {
	cases := []struct {
		url     string
		hosts   []string
		wantErr bool
	}{
		{"https://cdn.niuma007.com/a.exe", nil, false},
		{"http://cdn.niuma007.com/a.exe", nil, true},
		{"https://127.0.0.1/a.exe", nil, true},
		{"https://localhost/a.exe", nil, true},
		{"https://192.168.1.1/a.exe", nil, true},
		{"https://evil.com/a.exe", []string{"niuma007.com"}, true},
		{"https://cdn.niuma007.com/a.exe", []string{"niuma007.com"}, false},
		{"https://user:pass@cdn.niuma007.com/a.exe", nil, true},
	}
	for _, tc := range cases {
		err := validateDownloadURL(tc.url, tc.hosts)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: want error", tc.url)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: %v", tc.url, err)
		}
	}
}
