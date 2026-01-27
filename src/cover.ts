import AdmZip from 'adm-zip';
import path from 'path';
import mime from 'mime-types';
import fs from 'node:fs';

const SUPPORTED_ARCHIVES = ['.epub', '.cbz', '.zip', '.fb2.zip'];
const COVER_REGEX = /cover\.(jpe?g|png)/i;
const IMAGE_REGEX = /\.(jpe?g|png)$/i;

export const getCover = async (filePath: string): Promise<{ data: Buffer; mimeType: string } | null> => {
  try {
    const ext = path.extname(filePath).toLowerCase();

    if (SUPPORTED_ARCHIVES.includes(ext)) {
      const buffer = await fs.promises.readFile(filePath);
      const zip = new AdmZip(buffer);
      const entries = zip.getEntries();

      let entry = entries.find(e => COVER_REGEX.test(e.entryName));

      if (!entry) {
        const images = entries.filter(e => IMAGE_REGEX.test(e.entryName) && !e.entryName.includes('__MACOSX'));

        if (images.length > 0) {
          entry = images.find(e => e.entryName.toLowerCase().includes('cover')) || images[0];
        }
      }

      if (entry) {
        return {
          data: entry.getData(),
          mimeType: mime.lookup(entry.entryName) || 'image/jpeg'
        };
      }
    }

    return null;
  } catch (err) {
    console.error(err);
    return null;
  }
};
