import { createRequire } from 'node:module';
import { readFile, lstat, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { pathToFileURL } from 'node:url';

// This invocation-scoped adapter uses the already installed native attachment
// service. It never installs, modifies, or registers a persistent extension.
export async function apply(ctx, config) {
  const nativeRequire = createRequire(config.entry);
  const { createUserMessage } = await import(pathToFileURL(nativeRequire.resolve('@deepseek-ai/dsh-llm')).href);
  let delivered = false;
  ctx.on('agent/created', async ({ agent, signal }) => {
    if (delivered || agent.session.header.parentSession !== undefined) return;
    delivered = true;
    const images = config.images;
    const outcome = { delivered: [], references: [], note: 'A native image input is not a claim of understanding.' };
    const blocks = [];
    const routed = agent.session.requestHeader()?.config;
    const provider = routed?.provider ?? agent.options.provider;
    const model = routed?.model ?? agent.options.model;
    const llm = ctx.get('llm');
    const attachments = ctx.get('attachments');
    const route = provider && model && llm ? await llm.resolveModelInfo(provider, model, signal) : undefined;
    if (!route?.inputModalities?.includes('image') || !attachments) {
      outcome.references = images.map(image => ({ id: image.id, path: image.path, reason: 'Current route cannot inspect images.' }));
      blocks.push({ type: 'text', text: 'ATTACHMENT DELIVERY: The current model route cannot inspect images. The uploaded files remain available as file references, but do not describe their unseen visual contents. State this limitation to the user.' });
    } else {
      const inputs = [];
      const selected = [];
      const limits = attachments.imageLimits;
      let total = 0;
      for (let i = images.length - 1; i >= 0; i--) {
        const image = images[i];
        if (selected.length >= limits.maxImagesPerMessage || image.size > limits.maxImageBytes || image.size + total > limits.maxMessageImageBytes || !limits.mediaTypes.includes(image.media_type)) {
          outcome.references.unshift({ id: image.id, path: image.path, reason: 'Native image input limit; use an actual image tool if supported.' });
          continue;
        }
        const info = await lstat(image.path);
        if (!info.isFile() || info.isSymbolicLink() || info.size !== image.size) throw new Error('attachment_changed: native image source changed');
        const data = await readFile(image.path);
        if (data.length !== image.size || createHash('sha256').update(data).digest('hex') !== image.sha256) throw new Error('attachment_changed: native image digest changed');
        selected.unshift(image);
        inputs.unshift({ data, mediaType: image.media_type, name: image.name });
        total += data.length;
      }
      const refs = await attachments.saveImages(inputs);
      if (refs.length !== selected.length) throw new Error('attachment_delivery_invalid: native image admission returned an incomplete batch');
      blocks.push({ type: 'text', text: 'ATTACHMENT DELIVERY: The following uploaded images are supplied as actual native image parts. Other uploaded files remain references and must be read with tools before claiming their contents.' });
      for (let i = 0; i < refs.length; i++) {
        blocks.push({ type: 'text', text: `Image file reference: ${JSON.stringify(selected[i].path)}` });
        blocks.push({ type: 'image', attachment: refs[i] });
        outcome.delivered.push({ id: selected[i].id, path: selected[i].path, attachment: refs[i] });
      }
      if (outcome.references.length) blocks.push({ type: 'text', text: 'Images not attached as native images; inspect with a supported native image tool or report them unread: ' + JSON.stringify(outcome.references) });
    }
    await writeFile(config.receipt, JSON.stringify(outcome, null, 2), { mode: 0o600 });
    agent.inject(createUserMessage({ content: blocks, source: { kind: 'user' } }));
  });
}
