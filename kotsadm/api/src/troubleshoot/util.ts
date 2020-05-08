import tar from "tar-stream";
import zlib from "zlib";

export interface FilesAsBuffers {
  fakeIndex: string[];
  files: {
    [key: string]: Buffer;
  };
}

export interface RequestedFile {
  path: string;
  matcher: (file: string) => boolean;
}

export function isTgzByName(filenname: string): boolean {
  return filenname.endsWith('.tar.gz') || filenname.endsWith('.tgz');
}

export function eq(f1: string) {
  return (f2: string) => f1 === f2;
}

export function eqIgnoringLeadingSlash(f1: string) {
  let pathName = f1;
  if (f1.charAt(0) === "/") {
    pathName = f1.substring(1);
  }
  return (f2: string) => pathName === f2;
}

export function contains(f1: string) {
  return (f2: string) => f2.indexOf(f1) !== -1;
}

export function match(r: RegExp) {
  return (f2: string) => r.test(f2);
}

export class TarballUnpacker {

  /**
   * unpackFrom will read the stream
   */
  public async unpackFrom(tarStream: NodeJS.ReadableStream, filesWeCareAbout?: RequestedFile[]): Promise<FilesAsBuffers> {
    if (!tarStream) {
      return { files: {}, fakeIndex: [] };
    }

    return await new Promise<FilesAsBuffers>((resolve, reject) => {
      const extract = tar.extract();

      const extractedFiles: FilesAsBuffers = { files: {}, fakeIndex: [] };
      const promises: Array<Promise<any>> = [];

      extract.on("entry", (header: any, stream: NodeJS.ReadableStream, requestNextTarFile: () => void) => {
        let pathsToStoreThisTarEntry: string[];

        if (filesWeCareAbout) {
          pathsToStoreThisTarEntry = filesWeCareAbout.filter((f) => f.matcher(header.name)).map((f) => f.path);
        } else {
          pathsToStoreThisTarEntry = header.name;
        }

        const isFile = header.type === "file";
        if (isFile) {
          extractedFiles.fakeIndex.push(header.name);
        }
        const fileRequestedByAnalyzers = pathsToStoreThisTarEntry.length;
        if (isFile && fileRequestedByAnalyzers) {
          promises.push(this.readFile(stream).then((contents) => {
            for (const path of pathsToStoreThisTarEntry) {
              extractedFiles.files[path] = contents;
            }
          }));
        } else {
          stream.resume();
        }
        requestNextTarFile(); // we're reading in parallel, no need to block this

      });
      extract.on("error", reject);

      extract.on("finish", () => {
        Promise.all(promises).then(() => resolve(extractedFiles));
      });

      tarStream.pipe(extract);
    });

  }

  /**
   * this is kinda lazy to just read these all in, should
   * just pass streams around someday
   */
  private readFile(s: NodeJS.ReadableStream): Promise<Buffer> {
    return new Promise<Buffer>((resolve, reject) => {
      const buffers: Buffer[] = [];
      s.on("data", (chunk) => {
        buffers.push(chunk);
      });
      s.on("error", reject);
      s.on("end", () => {
        const contents = Buffer.concat(buffers);
        resolve(contents);
      });
    });
  }
}

export class TarballPacker {

  /**
   * packFiles will create a .tar.gz for files
   */
  public async packFiles(files: FilesAsBuffers): Promise<Buffer> {
    return new Promise(resolve => {
      const pack = tar.pack();
      for (const path in files.files) {
        pack.entry({ name: path }, files.files[path]);
      }
      const buffers: Buffer[] = [];
      const gzipStream = zlib.createGzip();
      pack.pipe(gzipStream)
        .on('data', (buffer: Buffer) => {
          buffers.push(buffer);
        })
        .on('end', () => {
          const tarGz = Buffer.concat(buffers);
          resolve(tarGz);
        });
      pack.finalize();
    })
  }
}
