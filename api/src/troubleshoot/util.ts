import * as tar from "tar-stream";

export interface FilesAsString {
  fakeIndex: string[];
  files: {
    [key: string]: string;
  };
}

export interface RequestedFile {
  path: string;
  matcher: (file: string) => boolean;
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
  public async unpackFrom(
    tarStream: NodeJS.ReadableStream,
    filesWeCareAbout: RequestedFile[],
  ): Promise<FilesAsString> {
    if (!tarStream) {
      return {files: {}, fakeIndex: []};
    }

    return await new Promise<FilesAsString>((resolve, reject) => {
      const extract = tar.extract();

      const extractedFiles: FilesAsString = {files: {}, fakeIndex: []};
      const promises: Array<Promise<any>> = [];

      extract.on("entry", (header: any, stream: NodeJS.ReadableStream, requestNextTarFile: () => void) => {

        const pathsToStoreThisTarEntry: string[] =
          filesWeCareAbout
            .filter((f) => f.matcher(header.name))
            .map((f) => f.path);

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
  private readFile(s: NodeJS.ReadableStream): Promise<string> {
    return new Promise<string>((resolve, reject) => {
      let contents = ``;
      s.on("data", (chunk) => {
        contents += chunk.toString();
      });
      s.on("error", reject);
      s.on("end", () => {
        resolve(contents);
      });
    });
  }
}