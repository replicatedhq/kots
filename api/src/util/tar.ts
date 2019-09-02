import { Parse as TarParser } from "tar";
import { PassThrough as PassThroughStream } from "stream";
import path from "path";
import * as _ from "lodash";
import * as tar from "tar-stream";
import * as zlib from "zlib";
import concat from "concat-stream";
import yaml from "js-yaml";

interface CursorAndVersion {
  cursor: string;
  versionLabel: string;
}

function bufferToStream(buffer: Buffer): NodeJS.ReadableStream {
  const stream = new PassThroughStream();
  stream.end(buffer);
  return stream;
}

export function extractCursorAndVersionFromTarball(tarball: Buffer): Promise<CursorAndVersion> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  return new Promise((resolve, reject) => {
    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat((data) => {
        const doc = yaml.safeLoad(data.toString());
        if (doc) {
          if ((doc.apiVersion === "kots.io/v1beta1") && (doc.kind === "Installation")) {
            console.log(doc);

            const cursorAndVersion = {
              cursor: doc.spec.updateCursor,
              versionLabel: doc.spec.versionLabel ? doc.spec.versionLabel : "??",
            };

            resolve(cursorAndVersion);
            next();
            return;
          }
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve({
        cursor: "",
        versionLabel: "",
      });
    });

    extract.end(uncompressed);
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
