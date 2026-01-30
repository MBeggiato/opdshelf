import * as fs from 'fs';
import * as path from 'path';
import Handlebars from 'handlebars';


export const renderView = async (viewName: string, data: any, layout = 'main') => {
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

export const renderXml = async (viewName: string, data: any) => {
  const source = await fs.promises.readFile(path.join(process.cwd(), 'views', `${viewName}.hbs`), 'utf8');
  return Handlebars.compile(source)(data);
}
