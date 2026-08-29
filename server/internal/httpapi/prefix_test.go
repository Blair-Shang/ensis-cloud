package httpapi

import "testing"

func TestPrefixes(t *testing.T) {
	if CloudPrefix != "/niuma/cloud" {
		t.Fatalf("cloud=%s", CloudPrefix)
	}
	if OpsPrefix != "/niuma/ops" {
		t.Fatalf("ops=%s", OpsPrefix)
	}
}

func TestIsCloudAPIPath(t *testing.T) {
	ok := []string{
		"/niuma/cloud/healthz",
		"/niuma/cloud/api/v1/me",
		"/niuma/cloud/api/v1/updates/hit",
		"/niuma/cloud/api/v1/updates/download",
	}
	for _, p := range ok {
		if !isCloudAPIPath(p) {
			t.Fatalf("want api path %s", p)
		}
	}
	no := []string{"/", "/niuma", "/niuma/ops", "/ops", "/cloud", "/cloud/healthz"}
	for _, p := range no {
		if isCloudAPIPath(p) {
			t.Fatalf("not api path %s", p)
		}
	}
}

func TestIsOpsPath(t *testing.T) {
	if !isOpsPath("/niuma/ops") || !isOpsPath("/niuma/ops/login") {
		t.Fatal("want ops paths")
	}
	if isOpsPath("/ops") || isOpsPath("/niuma/cloud/api/v1/me") {
		t.Fatal("must not match bare /ops or cloud API")
	}
}
