package httpapi

import "testing"

func TestCanonicalDesktopProduct(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "niuma"},
		{in: "niuma", want: "niuma"},
		{in: "ensis-ops", want: "ensis-ops"},
		{in: "ENSIS-OPS", want: "ensis-ops"},
		{in: "other", want: "other"},
	}
	for _, tc := range cases {
		got, _, _, _, err := normalizeReleaseDims(tc.in, "stable", "windows", "x64")
		if err != nil {
			t.Fatalf("product %q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("product %q: got %s want %s", tc.in, got, tc.want)
		}
	}
}
