import AdmZip from 'adm-zip';
import path from 'path';
import mime from 'mime-types';
import fs from 'node:fs';
import { XMLParser } from 'fast-xml-parser';

const SUPPORTED_ARCHIVES = ['.epub', '.cbz', '.zip', '.fb2.zip'];
const COVER_REGEX = /(^|\/)cover\.(jpe?g|png)$/i;
const IMAGE_REGEX = /\.(jpe?g|png)$/i;

export async function getEpubCover(filePath: string): Promise<{ data: Buffer; mimeType: string } | null> {
  try {
    const buffer = await fs.promises.readFile(filePath);
    const zip = new AdmZip(buffer);
    const parser = new XMLParser({ ignoreAttributes: false, attributeNamePrefix: "" });

    const containerEntry = zip.getEntry("META-INF/container.xml");
    if (!containerEntry) throw new Error("EPUB format no valid (missing container.xml)");

    const containerData = parser.parse(containerEntry.getData().toString());
    const opfPath = containerData.container?.rootfiles?.rootfile?.["full-path"];

    if (!opfPath) throw new Error("Could not find rootfile on container.xml");

    const opfEntry = zip.getEntry(opfPath);
    if (!opfEntry) throw new Error(`Could not find OPF file on ${opfPath}`);

    const opfContent = opfEntry.getData().toString();
    const opfData = parser.parse(opfContent);

    const manifest = opfData.package?.manifest?.item;
    const metadata = opfData.package?.metadata?.meta;

    if (!manifest) throw new Error("Could not find manifest");

    let coverHref = "";
    let coverMediaType = "";

    const items = Array.isArray(manifest) ? manifest : [manifest];
    const metas = Array.isArray(metadata) ? metadata : (metadata ? [metadata] : []);

    const coverMeta = metas.find((m: any) => m?.name === "cover");
    if (coverMeta) {
      const coverID = coverMeta.content;
      const item = items.find((i: any) => i.id === coverID);
      if (item) {
        coverHref = item.href;
        coverMediaType = item["media-type"];
      }
    }

    if (!coverHref) {
      const item = items.find((i: any) => i.properties === "cover-image");
      if (item) {
        coverHref = item.href;
        coverMediaType = item["media-type"];
      }
    }

    if (!coverHref) {
      const item = items.find((i: any) => i.id?.toLowerCase().includes("cover"));
      if (item) {
        coverHref = item.href;
        coverMediaType = item["media-type"];
      }
    }

    if (!coverHref) throw new Error("Could not find cover");

    const opfDir = path.dirname(opfPath);
    const fullImagePath = path.join(opfDir, coverHref).replace(/\\/g, "/");

    const imageEntry = zip.getEntry(fullImagePath);
    if (!imageEntry) throw new Error(`Could not find image: ${fullImagePath}`);

    return {
      data: imageEntry.getData(),
      mimeType: coverMediaType || "image/jpeg"
    };
  } catch (e) {
    return null;
  }
}

export const getCover = async (filePath: string): Promise<{ data: Buffer; mimeType: string } | null> => {
  try {
    const ext = path.extname(filePath).toLowerCase();

    if (ext === '.epub') {
      const epubCover = await getEpubCover(filePath);
      if (epubCover) return epubCover;
    }

    if (SUPPORTED_ARCHIVES.includes(ext)) {
      const buffer = await fs.promises.readFile(filePath);
      const zip = new AdmZip(buffer);
      const entries = zip.getEntries();

      let entry = entries.find(e => COVER_REGEX.test(e.entryName) && !e.isDirectory);

      if (!entry) {
        const images = entries.filter(e =>
          IMAGE_REGEX.test(e.entryName) &&
          !e.entryName.includes('__MACOSX') &&
          !e.isDirectory
        );

        if (images.length > 0) {
          images.sort((a, b) => {
            const sizeA = (a as any).header?.size || 0;
            const sizeB = (b as any).header?.size || 0;
            return sizeB - sizeA;
          });
          entry = images[0];
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
