import { Parse as TarParser } from "tar";
import { PassThrough as PassThroughStream } from "stream";
import path from "path";
import * as _ from "lodash";
import * as tar from "tar-stream";
import * as concat from "concat-stream";

function bufferToStream(buffer: Buffer): NodeJS.ReadableStream {
  const stream = new PassThroughStream();
  stream.end(buffer);
  return stream;
}

export function extractCursorFromTarball(tarball: Buffer): Promise<string> {
  const extract = tar.extract();

  return new Promise((resolve, reject) => {
    extract.on("entry", (header, stream, next) => {
      console.log(header);
      stream.pipe(concat((data) => {
        // const doc = yaml.safeLoad(data.toString());
        // if ((doc.apiVersion === "kots.io/v1beta1") && (doc.kind === "Application")) {
        //   resolve(data.toString());
        //   next();
        //   return;
        // }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve("");
    });

    extract.end(tarball);
  });
}

export function extractDownstreamNamesFromTarball(tarball: Buffer): Promise<string[]> {
  return new Promise<string[]>((resolve, reject) => {
    let downstreamNames: string[] = [];
    const parser = new TarParser({
      strict: true,
      filter: (currentPath: string) => {
        const parts = currentPath.split(path.sep);
        _.remove(parts, (n) => {
          return n.length === 0;
        });

        // the first part is always the name of the directory it was uploaded from
        if (parts.length === 4) {
          if (parts[0] === "overlays" && parts[1] === "downstreams" && parts[3] === "kustomization.yaml") {
            downstreamNames.push(parts[2]);
          }
        }
        return false;
      },
    });
    bufferToStream(tarball)
      .pipe(parser)
      .on('end', () => {
        resolve(downstreamNames);
      })
      .on('error', reject);
  });
}
