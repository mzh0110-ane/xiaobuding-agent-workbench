package reliable

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExistingNativeHarnessExtensionsResolveWithoutModelCalls(t *testing.T) {
	home, entry, node := os.Getenv("XIAOBUDING_NATIVE_PROFILE_HOME"), os.Getenv("XIAOBUDING_NATIVE_ENTRY"), os.Getenv("XIAOBUDING_NATIVE_NODE")
	if home == "" || entry == "" || node == "" {
		t.Skip("explicit native metadata verification is disabled")
	}
	profile := filepath.Join(home, "profiles", "desktop")
	files := []string{filepath.Join(profile, "package.json"), filepath.Join(profile, "cordis.patch.yml")}
	before := make(map[string][32]byte)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		before[file] = sha256.Sum256(raw)
	}
	runner := &HarnessRunner{Executable: node, Entry: entry, Home: home}
	out := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	patch, cleanup, err := runner.inheritedExtensions(ctx, RoleRequest{Directory: t.TempDir(), OutDir: out}, append(os.Environ(), "DSH_HOME="+home))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err = os.Stat(patch); err != nil {
		t.Fatal("the installed profile did not produce native references")
	}
	verifyNativeDisabledPreference(t, runner, patch)
	raw, err := os.ReadFile(filepath.Join(out, "native-extensions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct{ Bundles []string }
	if err = json.Unmarshal(raw, &inventory); err != nil || len(inventory.Bundles) == 0 {
		t.Fatal("no original extensions were referenced")
	}
	for _, file := range files {
		data, readErr := os.ReadFile(file)
		if readErr != nil || sha256.Sum256(data) != before[file] {
			t.Fatal("the existing native profile was modified")
		}
	}
	if destination := os.Getenv("XIAOBUDING_NATIVE_INVENTORY"); destination != "" {
		if err = os.WriteFile(destination, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("referenced %d existing native extension bundles without a model call or profile changes", len(inventory.Bundles))
}

func verifyNativeDisabledPreference(t *testing.T, runner *HarnessRunner, patch string) {
	t.Helper()
	id := os.Getenv("XIAOBUDING_NATIVE_DISABLED_ID")
	if id == "" {
		return
	}
	code := `const{createRequire}=require('node:module'),{pathToFileURL}=require('node:url');const[entry,patch,id]=process.argv.slice(1);(async()=>{const boot=await import(pathToFileURL(createRequire(entry).resolve('@deepseek-ai/dsh-app-boot')).href);const entries=boot.composeEntries([boot.loadOverlayPatches('native-check',patch)]);let row;function visit(items){for(const item of items){if(item.id===id)row=item;if(item.group&&Array.isArray(item.config))visit(item.config)}}visit(entries);console.log(JSON.stringify({found:!!row,disabled:row?.disabled===true}));})().catch(()=>process.exit(7));`
	stdout, _, _, err := runProcess(context.Background(), t.TempDir(), os.Environ(), "", runner.Executable, "-e", code, runner.Entry, patch, id)
	var result struct{ Found, Disabled bool }
	if err != nil || json.Unmarshal(stdout, &result) != nil || !result.Found || !result.Disabled {
		t.Fatal("the original named disabled preference was not preserved by native composition")
	}
	t.Logf("native composition preserved disabled preference %s", id)
}
