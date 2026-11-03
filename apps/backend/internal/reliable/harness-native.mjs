import { createRequire } from 'node:module';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { pathToFileURL } from 'node:url';

const [entry, home, overlayPath, inventoryPath] = process.argv.slice(2);
try {
  const nativeRequire = createRequire(entry);
  const bootPath = nativeRequire.resolve('@deepseek-ai/dsh-app-boot');
  const bootRequire = createRequire(bootPath);
  const boot = await import(pathToFileURL(bootPath).href);
  const profileDir = join(home, 'profiles', 'desktop');
  const desktopRequire = createRequire(join(profileDir, 'package.json'));
  const profile = boot.loadProfileDirectory('xiaobuding', profileDir, join(dirname(dirname(entry)), 'package.json'));
  if (profile.skippedBundles.length) {
    writeFileSync(inventoryPath, JSON.stringify({ status: 'unavailable', skipped_bundles: profile.skippedBundles.map(item => item.packageName) }, null, 2), { mode: 0o600 });
    throw new Error('native_profile_unavailable');
  }
  const excluded = new Set(['@deepseek-ai/dsh-base', '@deepseek-ai/dsh-web-app']);
  const headlessManifest = join(home, 'profiles', 'headless', 'package.json');
  if (existsSync(headlessManifest)) {
    for (const name of JSON.parse(readFileSync(headlessManifest, 'utf8')).dsh?.profile?.bundles || []) excluded.add(name);
  }
  const layers = profile.layers.filter(layer => !excluded.has(layer.packageName));
  const entries = [];
  const renamed = new Map();
  function rewriteEntry(item) {
    if (typeof item.name === 'string' && item.name && !item.name.startsWith('file:')) {
      const original = item.name;
      let resolved;
      try { resolved = desktopRequire.resolve(item.name); } catch { resolved = nativeRequire.resolve(item.name); }
      item.name = pathToFileURL(resolved).href;
      if (typeof item.id === 'string') renamed.set(item.id + '\0' + original, item.name);
    }
    if (item.name) entries.push({ id: item.id, module: item.name });
    if (item.group && Array.isArray(item.config)) item.config.forEach(rewriteEntry);
  }
  const patches = [...layers.flatMap(layer => layer.patches), ...profile.patches];
  for (const patch of patches) patch.insert?.forEach(rewriteEntry);
  for (const patch of patches) {
    const replacement = renamed.get(patch.id + '\0' + patch.name);
    if (replacement) patch.name = replacement;
  }
  const yaml = await import(pathToFileURL(bootRequire.resolve('js-yaml')).href);
  const include = await import(pathToFileURL(bootRequire.resolve('@deepseek-ai/cordis-plugin-include')).href);
  const dump = yaml.dump || yaml.default?.dump;
  const schema = include.entryListSchema || include.default?.entryListSchema;
  if (!schema || typeof dump !== 'function') throw new Error('native_serializer_unavailable');
  writeFileSync(overlayPath, dump(patches, { schema, noRefs: true }), { mode: 0o600 });
  writeFileSync(inventoryPath, JSON.stringify({ source: profileDir, host: 'headless', status: 'referenced', bundles: layers.map(layer => layer.packageName), entries, note: 'Original installed modules and preferences are reused. UI-only entries may wait for native host services.' }, null, 2), { mode: 0o600 });
} catch {
  process.stderr.write('Existing Harness extensions could not be prepared. Check the native desktop profile and installation.\n');
  process.exitCode = 1;
}
