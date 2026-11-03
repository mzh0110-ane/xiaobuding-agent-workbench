package reliable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func nativeAttachmentRequest(t *testing.T, count int) (RoleRequest, []string) {
	t.Helper()
	dir := t.TempDir()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 1))); err != nil {
		t.Fatal(err)
	}
	refs := make([]map[string]any, 0, count+1)
	paths := make([]string, 0, count)
	for i := range count {
		name := "图片 " + string(rune('一'+i)) + ".png"
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, imageBytes.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(imageBytes.Bytes())
		refs = append(refs, map[string]any{"id": "image-" + string(rune('a'+i)), "name": name, "path": path, "size": imageBytes.Len(), "media_type": "image/png", "sha256": hex.EncodeToString(hash[:]), "image": true})
		paths = append(paths, path)
	}
	note := filepath.Join(dir, "说明.txt")
	if err := os.WriteFile(note, []byte("document content"), 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("document content"))
	refs = append(refs, map[string]any{"id": "document", "name": "说明.txt", "path": note, "size": 16, "media_type": "text/plain", "sha256": hex.EncodeToString(hash[:]), "image": false})
	raw, err := json.Marshal(map[string]any{"attachments": refs})
	if err != nil {
		t.Fatal(err)
	}
	var req RoleRequest
	if err = json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	req.Role, req.Directory, req.OutDir, req.Prompt = roleAuditor, t.TempDir(), t.TempDir(), "Inspect the supplied images and file; report what you actually read."
	return req, paths
}

// @covers AC-RELIABLE-ATTACHMENTS-001.5
func TestNativeCodexReceivesActualImageArgumentsAndBoundedDelivery(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is unavailable")
	}
	req, paths := nativeAttachmentRequest(t, 10)
	preload := filepath.Join(t.TempDir(), "codex-images.cjs")
	script := `const fs=require('node:fs');const a=process.argv;const paths=[];for(let i=0;i<a.length;i++)if(a[i]==='--image')paths.push(a[++i]);fs.writeFileSync(process.env.ATTACHMENT_WIRE,JSON.stringify({paths}));fs.writeFileSync(process.env.ATTACHMENT_PROMPT,fs.readFileSync(0,'utf8'));fs.writeFileSync(a[a.indexOf('-o')+1],JSON.stringify({summary:'offline native images',read_files:[],docs_impact:'none',residual_risks:[]}));console.log(JSON.stringify({type:'thread.started',thread_id:'images-native'}));console.log(JSON.stringify({type:'turn.completed'}));process.exit(0);`
	if err = os.WriteFile(preload, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	wire, promptPath := filepath.Join(t.TempDir(), "wire.json"), filepath.Join(t.TempDir(), "prompt.txt")
	t.Setenv("NODE_OPTIONS", `--require="`+filepath.ToSlash(preload)+`"`)
	t.Setenv("ATTACHMENT_WIRE", wire)
	t.Setenv("ATTACHMENT_PROMPT", promptPath)
	runner := &CodexRunner{Executable: node, Home: t.TempDir(), Provider: "openai"}
	if _, err = runner.Run(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(wire)
	if err != nil {
		t.Fatal(err)
	}
	var received struct{ Paths []string }
	if err = json.Unmarshal(raw, &received); err != nil {
		t.Fatal(err)
	}
	if len(received.Paths) != 8 || strings.Join(received.Paths, "\n") != strings.Join(paths[2:], "\n") {
		t.Fatalf("native Codex did not receive the newest eight actual images: %s", raw)
	}
	prompt, err := os.ReadFile(promptPath)
	if err != nil || !strings.Contains(string(prompt), "not attached as native images") || !strings.Contains(string(prompt), strconv.Quote(paths[0])) {
		t.Fatalf("older images were silently discarded instead of declared as file references: %s %v", prompt, err)
	}
}

// @covers AC-RELIABLE-ATTACHMENTS-001.5
func TestNativeHarnessReceivesActualImageBlocksAndHonestFallback(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is unavailable")
	}
	for _, test := range []struct {
		name    string
		capable bool
		limit   string
		images  int
	}{{"image-capable", true, "20", 2}, {"native-lower-count-limit", true, "1", 1}, {"text-only", false, "20", 0}} {
		t.Run(test.name, func(t *testing.T) {
			req, _ := nativeAttachmentRequest(t, 2)
			home, entry := nativeInheritanceFixture(t)
			if err := os.WriteFile(entry, []byte(nativeImageCliFixture), 0600); err != nil {
				t.Fatal(err)
			}
			moduleDir := filepath.Join(filepath.Dir(filepath.Dir(entry)), "node_modules", "@deepseek-ai", "dsh-llm")
			if err := os.MkdirAll(moduleDir, 0700); err != nil {
				t.Fatal(err)
			}
			for name, body := range map[string]string{"package.json": `{"name":"@deepseek-ai/dsh-llm","main":"index.cjs"}`, "index.cjs": `exports.createUserMessage=x=>Object.freeze({...x,id:'offline-image-message',role:'user'});`} {
				if err := os.WriteFile(filepath.Join(moduleDir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			wire := filepath.Join(t.TempDir(), "wire.json")
			t.Setenv("ATTACHMENT_WIRE", wire)
			t.Setenv("ATTACHMENT_CAPABLE", map[bool]string{true: "yes", false: "no"}[test.capable])
			t.Setenv("ATTACHMENT_NATIVE_LIMIT", test.limit)
			runner := &HarnessRunner{Executable: node, Entry: entry, Home: home, Provider: "native-account", Model: "native-image-model"}
			if _, err := runner.Run(context.Background(), req); err != nil {
				diagnostic, _ := os.ReadFile(filepath.Join(req.OutDir, "stderr.txt"))
				t.Fatalf("native image bridge did not complete: %v %s", err, diagnostic)
			}
			raw, err := os.ReadFile(wire)
			if err != nil {
				t.Fatal(err)
			}
			var value struct {
				Images, Saves, Injections int
				Text                      string
			}
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			if test.capable && (value.Images != test.images || value.Saves != 1 || value.Injections != 1) {
				t.Fatalf("native model did not receive real immutable image parts once: %s", raw)
			}
			if !test.capable && (value.Images != 0 || value.Saves != 0 || !strings.Contains(value.Text, "cannot inspect images")) {
				t.Fatalf("text-only model falsely received a claim of seeing images: %s", raw)
			}
			if test.images == 1 && !strings.Contains(value.Text, "not attached as native images") {
				t.Fatalf("native count fallback omitted its unread file reference: %s", raw)
			}
		})
	}
}

const nativeImageCliFixture = `const fs=require('node:fs'),{fileURLToPath,pathToFileURL}=require('node:url');
const args=process.argv.slice(2);const patches=[];for(let i=0;i<args.length;i++)if(args[i]==='--patch')patches.push(JSON.parse(fs.readFileSync(args[++i],'utf8')));
const row=patches.flatMap(p=>p.flatMap(x=>x.insert||[])).find(x=>x.id==='xiaobuding-native-images');
if(!row){console.error('no native image bridge');process.exit(71);}
(async()=>{const bridge=await import(row.name);let created,saves=0;const messages=[];const ctx={get(name){return name==='attachments'?{imageLimits:{maxImageBytes:20971520,maxImagesPerMessage:Number(process.env.ATTACHMENT_NATIVE_LIMIT),maxMessageImageBytes:209715200,mediaTypes:['image/png','image/jpeg','image/webp','image/gif']},async saveImages(images){saves++;return images.map((x,i)=>({attachmentId:'sha256:'+i,mediaType:x.mediaType,bytes:x.data.byteLength,width:2,height:1,name:x.name}));}}:name==='llm'?{async resolveModelInfo(provider,model){if(provider!=='native-account'||model!=='native-image-model')throw Error('wrong route');return{inputModalities:process.env.ATTACHMENT_CAPABLE==='yes'?['text','image']:['text']};}}:undefined;},on(name,handler){if(name==='agent/created')created=handler;}};
await bridge.apply(ctx,row.config);if(!created)throw Error('image listener not installed');const agent={options:{provider:'native-account',model:'native-image-model'},session:{header:{},requestHeader(){return undefined;}},inject(message){messages.push(message);}};await created({agent});await created({agent});const content=messages.flatMap(m=>m.content);fs.writeFileSync(process.env.ATTACHMENT_WIRE,JSON.stringify({images:content.filter(x=>x.type==='image'&&x.attachment?.attachmentId).length,saves,injections:messages.length,text:content.filter(x=>x.type==='text').map(x=>x.text).join('\n')}));
let input='';process.stdin.on('data',x=>input+=x);process.stdin.on('end',()=>{for(const e of[{type:'session',sessionId:'harness-image-native',cwd:process.cwd()},{type:'status',phase:'turn_end',reason:{kind:'completed'}},{type:'final',text:JSON.stringify({summary:'offline native image',read_files:[],docs_impact:'none',residual_risks:[]})}])console.log(JSON.stringify(e));});process.stdin.resume();})().catch(e=>{console.error(e);process.exit(72)});`
