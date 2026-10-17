package config

import (
	"os"
	"testing"

	relaycfg "github.com/GrayCodeAI/relay/config"
)

func TestSetGatewayRegion_XiaomiClearsStaleBaseHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	t.Setenv("RELAY_CONFIG_DIR", dir)
	t.Setenv("XIAOMI_MIMO_TOKEN_PLAN_BASE_URL", "https://caller-owned.example.test/v1")
	cfg := &relaycfg.ProviderConfig{
		Version:                    "1",
		XiaomiMimoTokenPlanRegion:  "cn",
		XiaomiMimoTokenPlanBaseURL: "https://token-plan-cn.xiaomimimo.com/v1",
	}
	if err := relaycfg.SaveProviderConfig(cfg, ""); err != nil {
		t.Fatal(err)
	}
	if err := SetGatewayRegion(ProviderXiaomiTokenPlan, "sgp"); err != nil {
		t.Fatal(err)
	}
	loaded := relaycfg.LoadProviderConfig("")
	if loaded.XiaomiMimoTokenPlanRegion != "sgp" {
		t.Fatalf("region = %q", loaded.XiaomiMimoTokenPlanRegion)
	}
	if got := os.Getenv("XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"); got != "https://caller-owned.example.test/v1" {
		t.Fatalf("SetGatewayRegion mutated process env: %q", got)
	}
}

func TestNeedsGatewayRegion_XiaomiInvalidAndMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REFLECT_CONFIG_DIR", dir)
	t.Setenv("RELAY_CONFIG_DIR", dir)

	if !NeedsGatewayRegion(ProviderXiaomiTokenPlan) {
		t.Fatal("expected true when no config file")
	}
	if err := relaycfg.SaveProviderConfig(&relaycfg.ProviderConfig{Version: "1", XiaomiMimoTokenPlanRegion: "tokyo"}, ""); err != nil {
		t.Fatal(err)
	}
	if !NeedsGatewayRegion(ProviderXiaomiTokenPlan) {
		t.Fatal("expected true for invalid region")
	}
	_ = SetGatewayRegion(ProviderXiaomiTokenPlan, "cn")
	if NeedsGatewayRegion(ProviderXiaomiTokenPlan) {
		t.Fatal("expected false after valid region set")
	}
}
