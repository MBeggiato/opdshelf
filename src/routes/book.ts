import { getCover } from "../helpers/cover";
import mime from 'mime-types';
import * as fs from 'fs';
import * as path from 'path';
import { Hono } from "hono";
import { getConfig } from '../config';

const app = new Hono();
const config = getConfig();

app.get('/:filename', async (c) => {
  const filePath = path.join(config.BOOKS_DIR, c.req.param('filename'));

  console.log(filePath);
  if (fs.existsSync(filePath)) {
    const stat = fs.statSync(filePath);
    c.header('Content-Type', mime.lookup(filePath) || 'application/octet-stream');
    c.header('Content-Length', stat.size.toString());
    return c.body(fs.createReadStream(filePath) as any);
  }
  return c.notFound();
});

app.post('/upload', async (c) => {
  const body = await c.req.parseBody();
  const file = body['book'];

  if (file instanceof File) {
    if (!file.name) return c.text('No filename', 400);

    const dest = path.join(config.BOOKS_DIR, file.name);
    console.log(`Uploading ${file.name}`);

    try {
      await fs.promises.writeFile(dest, Buffer.from(await file.arrayBuffer()));
    } catch (err) {
      console.error(err);
      return c.text('Upload failed', 500);
    }
  }

  return c.redirect('/admin');
});

app.post('/delete/:filename', async (c) => {
  const filePath = path.join(config.BOOKS_DIR, path.basename(c.req.param('filename')));

  try {
    if (fs.existsSync(filePath)) await fs.promises.unlink(filePath);
  } catch (err) {
    console.error(err);
  }

  return c.redirect('/admin');
});

app.post('/rename', async (c) => {
  const { oldFilename, newFilename } = await c.req.parseBody();

  if (oldFilename && newFilename) {
    const safeOld = path.basename(oldFilename as string);
    let safeNew = path.basename(newFilename as string);

    if (!path.extname(safeNew)) safeNew += path.extname(safeOld);

    const oldPath = path.join(config.BOOKS_DIR, safeOld);
    const newPath = path.join(config.BOOKS_DIR, safeNew);

    try {
      if (fs.existsSync(oldPath) && !fs.existsSync(newPath)) {
        await fs.promises.rename(oldPath, newPath);
      }
    } catch (err) {
      console.error(err);
    }
  }

  return c.redirect('/admin');
});

app.get('/cover/:filename', async (c) => {
  const filePath = path.join(config.BOOKS_DIR, c.req.param('filename'));
  if (!fs.existsSync(filePath)) return c.notFound();

  const cover = await getCover(filePath);
  if (cover) {
    c.header('Content-Type', cover.mimeType);
    c.header('Cache-Control', 'public, max-age=86400');
    return c.body(cover.data as any);
  }

  return c.notFound();
});


export default app;
