import zlib from "zlib";
import tar from "tar-stream";
import fs from "fs";
import path from "path";
import mkdirp from "mkdirp";

export async function extractFromTgzStream(tgzStream: any, dstDir: string): Promise<void> {
    const extract = tar.extract();
    const gzunipStream = zlib.createGunzip();

    return new Promise((resolve, reject) => {
      extract.on("entry", async (header, stream, next) => {
        if (header.type !== "file") {
          stream.resume();
          next();
          return;
        }

        const fileName = path.join(dstDir, header.name);

        const parsed = path.parse(fileName);
        if (!fs.existsSync(parsed.dir)) {
            // TODO, move to node 10 and use the built in
            // fs.mkdirSync(parsed.dir, {recursive: true});
            mkdirp.sync(parsed.dir);
        }
        var fileWriter = fs.createWriteStream(fileName);
        stream.pipe(fileWriter);
        next();
      });

      extract.on("finish", () => {
          resolve();
      });

      tgzStream.pipe(gzunipStream).pipe(extract);
    });
}
