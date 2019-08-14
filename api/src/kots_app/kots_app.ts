import { Params } from "../server/params";
import zlib from "zlib";
import { eq, eqIgnoringLeadingSlash, FilesAsString, TarballUnpacker } from "../troubleshoot/util";
import { getS3 } from "../util/s3";
import _ from "lodash";
import { logger } from "../server/logger";

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

  async downloadFiles(watchId: string, sequence: string, filesWeCareAbout: Array<{ path: string; matcher }>): Promise<FilesAsString> {
    const replicatedParams = await Params.getParams();

    return new Promise<FilesAsString>((resolve, reject) => {
      const params = {
        Bucket: replicatedParams.shipOutputBucket,
        Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}${watchId}/${sequence}.tar.gz`,
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

  public toSchema() {
    return {
      ...this
    }
  }
}
