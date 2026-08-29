package releasefiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	d, err := Normalize(Dim{
		Product: "NiuMa", Channel: "Stable", Platform: "Windows",
		Arch: "X64", Version: "1.0.2", Name: "NiuMa-1.0.2-windows-x64-Setup.exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Product != "niuma" || d.Platform != "windows" {
		t.Fatalf("%+v", d)
	}
	if _, err := Normalize(Dim{Product: "niuma", Channel: "stable", Platform: "windows", Arch: "x64", Version: "1.0.2", Name: "../x.exe"}); err == nil {
		t.Fatal("want reject traversal")
	}
}

func TestSaveAndPublicURL(t *testing.T) {
	root := t.TempDir()
	st := New(root)
	payload := []byte("installer-bytes")
	sum := sha256.Sum256(payload)
	d := Dim{
		Product: "niuma", Channel: "stable", Platform: "windows",
		Arch: "x64", Version: "1.0.2", Name: "NiuMa-1.0.2-windows-x64-Setup.exe",
	}
	n, got, err := st.Save(d, bytes.NewReader(payload), 1024, hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("n=%d sum=%s", n, got)
	}
	if !st.Exists(d) {
		t.Fatal("missing file")
	}
	u := PublicURL("https://www.niuma007.com/niuma/cloud", d)
	if !strings.Contains(u, "/api/v1/updates/files/niuma/stable/windows/x64/1.0.2/") {
		t.Fatalf("url=%s", u)
	}

	payload2 := []byte("installer-bytes-v2")
	if _, _, err := st.Save(d, bytes.NewReader(payload2), 1024, ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(st.FilePath(d))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(payload2) {
		t.Fatal("overwrite failed")
	}

	found, ok := st.Resolve(Dim{
		Product: "niuma", Channel: "stable", Platform: "windows",
		Arch: "x64", Version: "1.0.2",
	})
	if !ok || found.Name != d.Name {
		t.Fatalf("resolve=%+v ok=%v", found, ok)
	}
}
