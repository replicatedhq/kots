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

export function getImageFormats(rootDir: string): string[] {
  // top level folders are image format names

  var imageFormats: string[] = [];

  fs.readdirSync(rootDir).forEach(file => {
    const fullPath = path.join(rootDir, file);
    const fileStat = fs.statSync(fullPath);
    if (fileStat.isDirectory()) {
      imageFormats.push(file);
    }
  });

  return imageFormats
}

export function getImageFiles(rootDir: string): string[] {
  var imageFiles: string[] = [];

  fs.readdirSync(rootDir).forEach(file => {
    const fullPath = path.join(rootDir, file);
    const fileStat = fs.statSync(fullPath);
    if (fileStat.isDirectory()) {
      const files = getImageFiles(fullPath);
      imageFiles = imageFiles.concat(files);
    } else if (fileStat.isFile()) {
      imageFiles.push(fullPath);
    } else {
      // maybe it's a link, but our archives should only have files and folders
    }
  });

  return imageFiles
}

export function pathToImageName(rootDir: string, filePath: string): string {
  // filePath is /tmp/akjsdh/images/<format>/docker.io/couches/redis/2.0
  // rootDir is everything up to docker.io (registry host)
  // first turn it into docker.io/couches/redis/2.0
  // then extract image name and tag from the end of the string

  filePath = filePath.replace(rootDir, "");
  if (filePath[0] === path.sep) {
    filePath = filePath.slice(1);
  }

  var pathParts = filePath.split(path.sep);
  if (pathParts[0] === "docker.io") {
    pathParts.shift();
  }
  if (pathParts[0] === "library") {
    pathParts.shift();
  }

  var separator = ":";
  var tag = pathParts.pop();
  if (pathParts[pathParts.length - 1] === "sha256") {
    pathParts.pop();
    tag = "sha256:" + tag;
    separator = "@";
  }

  const imageName = pathParts.join("/"); // this is a url slash, not fs slash

  return imageName + separator + tag;
}

export function pathToShortImageName(rootDir: string, filePath: string): string {
  filePath.replace(rootDir, "");
  var pathParts = filePath.split(path.sep); // no path.split(filePath);

  var separator = ":";
  var tag = pathParts.pop();
  if (pathParts[pathParts.length - 1] === "sha256") {
    pathParts.pop();
    tag = "sha256:" + tag;
    separator = "@";
  }

  const imageName = pathParts.pop()

  return imageName + separator + tag;
}