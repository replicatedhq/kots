import { Params } from "../server/params";
import zlib from "zlib";
import { eq, eqIgnoringLeadingSlash, FilesAsString, TarballUnpacker } from "../troubleshoot/util";
import { getS3 } from "../util/s3";
import { logger } from "../server/logger";
import tmp from "tmp";
import fs from "fs";
import path from "path";
import tar from "tar-stream";
import mkdirp from "mkdirp";
import { exec } from "child_process";
import { Cluster } from "../cluster";
import * as _ from "lodash";

export class KotsApp {
  id: string;
  name: string;
  iconUri: string;
  createdAt: Date;
  updatedAt?: Date;
  slug: string;
  currentSequence?: number;
  lastUpdateCheckAt?: Date;

  // Source files
  async generateFileTreeIndex(sequence) {
    const supportBundleIndexJsonPath = "index.json";
    const indexFiles = await this.downloadFiles(this.id, sequence, [{
      path: supportBundleIndexJsonPath,
      matcher: eq(supportBundleIndexJsonPath),
    }]);

    const index = indexFiles.files[supportBundleIndexJsonPath] &&
      JSON.parse(indexFiles.files[supportBundleIndexJsonPath]);

    let paths: string[] = [];
    if (!index) {
      paths = indexFiles.fakeIndex;
    } else {
      index.map((p) => (paths.push(p.path)));
    }

    const dirTree = await this.arrangeIntoTree(paths);
    return dirTree;
  }

  arrangeIntoTree(paths) {
    const tree: any[] = [];
    _.each(paths, (path) => {
      const pathParts = path.split("/");
      if (pathParts[0] === "") {
        pathParts.shift(); // remove first blank element from the parts array.
      }
      let currentLevel = tree; // initialize currentLevel to root
      _.each(pathParts, (part) => {
        // check to see if the path already exists.
        const existingPath = _.find(currentLevel, ["name", part]);
        if (existingPath) {
          // the path to this item was already in the tree, so don't add it again.
          // set the current level to this path's children
          currentLevel = existingPath.children;
        } else {
          const newPart = {
            name: part,
            path: `${path}`,
            children: [],
          };
          currentLevel.push(newPart);
          currentLevel = newPart.children;
        }
      });
    });
    return tree;
  }

  async getFiles(sequence: string, fileNames: string[]): Promise<FilesAsString> {
    const fileNameList = fileNames.map((fileName) => ({
      path: fileName,
      matcher: eqIgnoringLeadingSlash(fileName),
    }));
    const filesWeWant = await this.downloadFiles(this.id, sequence, fileNameList);
    return filesWeWant;
  }

  async downloadFiles(appId: string, sequence: string, filesWeCareAbout: Array<{ path: string; matcher }>): Promise<FilesAsString> {
    const replicatedParams = await Params.getParams();

    return new Promise<FilesAsString>((resolve, reject) => {
      const params = {
        Bucket: replicatedParams.shipOutputBucket,
        Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}${appId}/${sequence}.tar.gz`,
      };
      logger.info({ msg: "S3 Params", params });

      const tarGZStream = getS3(replicatedParams).getObject(params).createReadStream();

      tarGZStream.on("error", reject);
      const unzipperStream = zlib.createGunzip();
      unzipperStream.on("error", reject);
      tarGZStream.pipe(unzipperStream);

      const bundleUnpacker = new TarballUnpacker();
      bundleUnpacker.unpackFrom(unzipperStream, filesWeCareAbout)
        .then(resolve)
        .catch(reject);
    });
  }

  async getArchive(sequence: string): Promise<any> {
    const replicatedParams = await Params.getParams();
    const params = {
      Bucket: replicatedParams.shipOutputBucket,
      Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}${this.id}/${sequence}.tar.gz`,
    };
    logger.info({ msg: "S3 Params", params });

    const result = await getS3(replicatedParams).getObject(params).promise();
    return result.Body;
  }

  async render(sequence: string, overlayPath: string): Promise<string> {
    const replicatedParams = await Params.getParams();
    const tmpDir = tmp.dirSync();

    try {
      const params = {
        Bucket: replicatedParams.shipOutputBucket,
        Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}${this.id}/${sequence}.tar.gz`,
      };
      logger.info({ msg: "S3 Params", params });

      const tgzStream = getS3(replicatedParams).getObject(params).createReadStream();
      const extract = tar.extract();
      const gzunipStream = zlib.createGunzip();

      return new Promise((resolve, reject) => {
        extract.on("entry", async (header, stream, next) => {
          if (header.type !== "file") {
            stream.resume();
            next();
            return;
          }

          const contents = await this.readFile(stream);

          const fileName = path.join(tmpDir.name, header.name);

          const parsed = path.parse(fileName);
          if (!fs.existsSync(parsed.dir)) {
            // TODO, move to node 10 and use the built in
            // fs.mkdirSync(parsed.dir, {recursive: true});
            mkdirp.sync(parsed.dir);
          }

            fs.writeFileSync(fileName, contents);
          next();
        });

        extract.on("finish", () => {
          // Run kustomize
          exec(`kustomize build ${path.join(tmpDir.name, overlayPath)}`, (err, stdout, stderr) => {
            if (err) {
              logger.error({msg: "err running kustomize", err, stderr})
              reject(err);
              return;
            }

            resolve(stdout);
          });
        });

        tgzStream.pipe(gzunipStream).pipe(extract);
      });

    } finally {
      // tmpDir.removeCallback();
    }
  }

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

  public toSchema(downstreams: Cluster[]) {
    return {
      ...this,
      downstreams: _.map(downstreams, (downstream) => {
        return {
          name: downstream.title,
          cluster: {
            ...downstream,
          },
        };
      }),
    };
  }
}
