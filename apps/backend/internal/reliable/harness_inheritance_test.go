package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessReusesInstalledDesktopExtensionsAndKeepsPreferencesPrivate(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node runtime is unavailable")
	}
	home, entry := nativeInheritanceFixture(t)
	out := t.TempDir()
	runner := &HarnessRunner{Executable: node, Entry: entry, Home: home, Provider: "deepseek-account", Model: "native"}
	result, err := runner.Run(context.Background(), RoleRequest{Role: roleAuditor, Directory: t.TempDir(), OutDir: out, Prompt: "Read source and return JSON"})
	if err != nil || result == nil || result.Session != "inherited-native" {
		t.Fatalf("the installed native extension was not inherited: %+v %v", result, err)
	}
	metadata, err := os.ReadFile(filepath.Join(out, "native-extensions.json"))
	if err != nil || !strings.Contains(string(metadata), "installed-skill") {
		t.Fatalf("native inheritance has no source inventory: %s %v", metadata, err)
	}
	if err = filepath.WalkDir(out, func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil || item.IsDir() {
			return walkErr
		}
		raw, readErr := os.ReadFile(path)
		if strings.Contains(string(raw), "private-fixture-preference") {
			t.Fatalf("native private configuration was copied into task evidence: %s", path)
		}
		return readErr
	}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(out, "stderr.txt"))
	patch := strings.TrimSpace(strings.TrimPrefix(string(raw), "private-patch:"))
	if _, err = os.Stat(patch); !os.IsNotExist(err) {
		t.Fatal("the invocation's private overlay was retained after completion")
	}
}

func nativeInheritanceFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "existing-home")
	native := filepath.Join(root, "native")
	profile := filepath.Join(home, "profiles", "desktop")
	files := map[string]string{
		filepath.Join(profile, "package.json"):                                                         `{"dsh":{"profile":{"bundles":["@deepseek-ai/dsh-base","@deepseek-ai/dsh-web-app","installed-skill"]}}}`,
		filepath.Join(profile, "cordis.patch.yml"):                                                     `[{"id":"native-skill","name":"installed-skill","disabled":true,"config":{"private":"private-fixture-preference"}}]`,
		filepath.Join(profile, "node_modules", "installed-skill", "package.json"):                      `{"name":"installed-skill","type":"module","exports":{".":"./index.mjs","./helper":"./helper.mjs"},"dsh":{"bundle":{"patch":"cordis.patch.yml"}}}`,
		filepath.Join(profile, "node_modules", "installed-skill", "index.mjs"):                         `throw new Error('metadata resolution must not execute plugins');`,
		filepath.Join(profile, "node_modules", "installed-skill", "helper.mjs"):                        `throw new Error('metadata resolution must not execute plugins');`,
		filepath.Join(profile, "node_modules", "installed-skill", "cordis.patch.yml"):                  `[{"insert":[{"id":"native-skill","name":"installed-skill","config":{"enabled":true}},{"id":"native-group","group":true,"config":[{"id":"native-helper","name":"installed-skill/helper"}]}]}]`,
		filepath.Join(native, "package.json"):                                                          `{"name":"existing-native-runtime","type":"commonjs"}`,
		filepath.Join(native, "node_modules", "@deepseek-ai", "dsh-app-boot", "package.json"):          `{"name":"@deepseek-ai/dsh-app-boot","main":"index.cjs"}`,
		filepath.Join(native, "node_modules", "@deepseek-ai", "dsh-app-boot", "index.cjs"):             nativeProfileLoaderFixture,
		filepath.Join(native, "node_modules", "@deepseek-ai", "cordis-plugin-include", "package.json"): `{"name":"@deepseek-ai/cordis-plugin-include","main":"index.cjs"}`,
		filepath.Join(native, "node_modules", "@deepseek-ai", "cordis-plugin-include", "index.cjs"):    `exports.entryListSchema={};`,
		filepath.Join(native, "node_modules", "js-yaml", "package.json"):                               `{"name":"js-yaml","main":"index.cjs"}`,
		filepath.Join(native, "node_modules", "js-yaml", "index.cjs"):                                  `exports.dump=(value)=>JSON.stringify(value);`,
		filepath.Join(native, "lib", "bin.cjs"):                                                        nativeInheritedCliFixture,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return home, filepath.Join(native, "lib", "bin.cjs")
}

func TestHarnessInheritanceFailureStopsBeforeCallingTheModel(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node runtime is unavailable")
	}
	home, entry := nativeInheritanceFixture(t)
	loader := filepath.Join(filepath.Dir(filepath.Dir(entry)), "node_modules", "@deepseek-ai", "dsh-app-boot", "index.cjs")
	if err = os.WriteFile(loader, []byte(`exports.loadProfileDirectory=()=>{throw new Error('private-fixture-preference')}`), 0600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	runner := &HarnessRunner{Executable: node, Entry: entry, Home: home}
	result, err := runner.Run(context.Background(), RoleRequest{Role: roleAuditor, Directory: t.TempDir(), OutDir: out})
	if err == nil || result != nil || strings.Contains(err.Error(), "private-fixture-preference") {
		t.Fatalf("a broken native profile did not stop safely: %v", err)
	}
	if _, err = os.Stat(filepath.Join(out, "events.jsonl")); !os.IsNotExist(err) {
		t.Fatal("the native model process ran after extension preparation failed")
	}
}

const nativeProfileLoaderFixture = `const fs=require('node:fs'),path=require('node:path');
exports.loadProfileDirectory=(bin,dir,anchor)=>({layers:[{packageName:'@deepseek-ai/dsh-base',patches:[]},{packageName:'@deepseek-ai/dsh-web-app',patches:[]},{packageName:'installed-skill',packageDir:path.join(dir,'node_modules/installed-skill'),patches:JSON.parse(fs.readFileSync(path.join(dir,'node_modules/installed-skill/cordis.patch.yml'),'utf8'))}],patches:JSON.parse(fs.readFileSync(path.join(dir,'cordis.patch.yml'),'utf8')),skippedBundles:[]});`

const nativeInheritedCliFixture = `const fs=require('node:fs'),{fileURLToPath}=require('node:url');const args=process.argv.slice(2);const patches=[];for(let i=0;i<args.length;i++)if(args[i]==='--patch')patches.push(args[i+1]);
if(patches.length!==2)process.exit(4);const inherited=JSON.parse(fs.readFileSync(patches[0],'utf8'));const skill=inherited.find(p=>p.insert)?.insert[0],group=inherited.find(p=>p.insert)?.insert[1];const preference=inherited.find(p=>p.id==='native-skill');if(!skill?.name.startsWith('file:')||!fs.existsSync(fileURLToPath(skill.name))||!group.config[0].name.startsWith('file:')||preference?.disabled!==true||preference.name!==skill.name||preference.config.private!=='private-fixture-preference')process.exit(5);const policy=JSON.parse(fs.readFileSync(patches[1],'utf8'));if(policy.find(p=>p.id==='sandbox-policy').config.mode!=='danger-full-access')process.exit(6);
let input='';process.stdin.on('data',x=>input+=x);process.stdin.on('end',()=>{console.error('private-patch:'+patches[0]);for(const event of [{type:'session',sessionId:'inherited-native',cwd:process.cwd()},{type:'status',phase:'turn_end',reason:{kind:'completed'}},{type:'final',text:JSON.stringify({summary:'reused native extension',read_files:[],docs_impact:'none',residual_risks:[]})}])console.log(JSON.stringify(event));});`
