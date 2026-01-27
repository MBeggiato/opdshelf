import { Hono } from 'hono';
import { serveStatic } from 'hono/bun';
import fs from 'node:fs';
import path from 'node:path';
import mime from 'mime-types';
import Handlebars from 'handlebars';
import { getConfig } from './config';
import { getBooks, sortBooks, formatSize, getSimpleMime } from './utils';
import { getCover } from './cover';
import { SortMode } from './types';

const app = new Hono();
const config = getConfig();

Handlebars.registerHelper('formatSize', formatSize);
Handlebars.registerHelper('simpleMime', getSimpleMime);
Handlebars.registerHelper('eq', (a, b) => a === b);
Handlebars.registerHelper('or', (a, b) => a || b);
Handlebars.registerHelper('len', (arr: any[]) => arr?.length || 0);

const renderView = async (viewName: string, data: any, layout = 'main') => {
  try {
    const cwd = process.cwd();
    const [viewSource, layoutSource] = await Promise.all([
      fs.promises.readFile(path.join(cwd, 'views', `${viewName}.hbs`), 'utf8'),
      fs.promises.readFile(path.join(cwd, 'views', 'layouts', `${layout}.hbs`), 'utf8')
    ]);

    const content = Handlebars.compile(viewSource)(data);
    return Handlebars.compile(layoutSource)({ ...data, body: content });
  } catch (e: any) {
    console.error(e);
    return e.message;
  }
};

const renderXml = async (viewName: string, data: any) => {
  const source = await fs.promises.readFile(path.join(process.cwd(), 'views', `${viewName}.hbs`), 'utf8');
  return Handlebars.compile(source)(data);
}

app.get('/', async (c) => {
  const books = await getBooks(config.BOOKS_DIR);
  const sortMode = (c.req.query('sort') as SortMode) || 'date-desc';

  const xml = await renderXml('opds', {
    books: sortBooks(books, sortMode),
    baseUrl: getBaseUrl(c, config),
    currentTime: new Date().toISOString(),
    sortMode
  });

  c.header('Content-Type', 'application/atom+xml;charset=utf-8;profile=opds-catalog;kind=acquisition');
  return c.body(xml);
});

app.get('/admin', async (c) => {
  const books = await getBooks(config.BOOKS_DIR);
  const sortMode = (c.req.query('sort') as SortMode) || 'date-desc';

  const html = await renderView('admin', {
    books: sortBooks(books, sortMode),
    baseUrl: getBaseUrl(c, config),
    sortMode,
    title: 'OPDShelf Admin'
  });

  return c.html(html);
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

app.use('/books/*', (c, next) => next());
app.get('/books/:filename', async (c) => {
  const filePath = path.join(config.BOOKS_DIR, c.req.param('filename'));

  if (fs.existsSync(filePath)) {
    const stat = fs.statSync(filePath);
    c.header('Content-Type', mime.lookup(filePath) || 'application/octet-stream');
    c.header('Content-Length', stat.size.toString());
    return c.body(fs.createReadStream(filePath) as any);
  }
  return c.notFound();
});

app.use('/static/*', serveStatic({ root: './' }));

function getBaseUrl(c: any, cfg: any) {
  if (cfg.REVERSE_PROXY) return cfg.REVERSE_PROXY_HOST;
  return `${c.req.header('x-forwarded-proto') || 'http'}://${c.req.header('host')}`;
}

export default app;
