import _ from "lodash";
import zlib from "zlib";
import { eq, eqIgnoringLeadingSlash, FilesAsString, TarballUnpacker } from "./util";
import { getS3 } from "../util/s3";
import { Params } from "../server/params";

export type SupportBundleStatus = "pending" | "uploaded" | "analyzing" | "analyzed" | "analysis_error";

export interface SupportBundleUpload {
  uploadUri: string;
  supportBundle: SupportBundle;
}

export class SupportBundle {
  id: string;
  slug: string;
  watchId: string;
  name: string;
  size: number;
  status: SupportBundleStatus;
  treeIndex: string;
  createdAt: Date;
  uploadedAt: Date;
  isArchived: boolean;
  analysis: SupportBundleAnalysis;
  watchSlug: string;
  watchName: string;

  async generateFileTreeIndex() {
    const supportBundleIndexJsonPath = "index.json";
    const indexFiles = await this.downloadFiles(this.id, [{
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

  async getFiles(bundle: SupportBundle, fileNames: string[]): Promise<FilesAsString> {
    const fileNameList = fileNames.map((fileName) => ({
      path: fileName,
      matcher: eqIgnoringLeadingSlash(fileName),
    }));
    const filesWeWant = await this.downloadFiles(bundle.id, fileNameList);
    return filesWeWant;
  }

  public static isS3NotFoundError(err) {
    return (
      err.code === "NoSuchKey" ||
      err.code === "AccessDenied" ||
      err.code === "NotFound" ||
      err.code === "Forbidden"
    );
  }

  async downloadFiles(bundleId: string, filesWeCareAbout: Array<{ path: string; matcher }>): Promise<FilesAsString> {
    const replicatedParams = await Params.getParams();

    return new Promise<FilesAsString>((resolve, reject) => {
      const params = {
        Bucket: replicatedParams.shipOutputBucket,
        Key: `${replicatedParams.s3BucketEndpoint !== "" ? replicatedParams.shipOutputBucket + "/" : ""}supportbundles/${bundleId}/supportbundle.tar.gz`,
      };

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
      id: this.id,
      slug: this.slug,
      name: this.name,
      size: this.size,
      status: this.status,
      treeIndex: this.treeIndex,
      createdAt: this.createdAt ? this.createdAt.toISOString() : undefined,
      uploadedAt: this.uploadedAt ? this.uploadedAt.toISOString() : undefined,
      isArchived: this.isArchived,
      analysis: this.analysis ? this.analysis.toSchema() : undefined
    };
  }
};

export class SupportBundleAnalysis {
  id: string;
  error: string;
  maxSeverity: string;
  insights: SupportBundleInsight[];
  createdAt: Date;

  public toSchema() {
    return {
      id: this.id,
      error: this.error,
      maxSeverity: this.maxSeverity,
      insights: _.map(this.insights, (insight) => {
        return insight.toSchema();
      }),
      createdAt: this.createdAt ? this.createdAt.toISOString() : undefined,
    };
  }
};

export class SupportBundleInsight {
  key: string;
  severity: string;
  primary: string;
  detail: string;
  icon: string;
  iconKey: string;
  desiredPosition: number;

  public toSchema() {
    return {
      key: this.key,
      severity: this.severity,
      primary: this.primary,
      detail: this.detail,
      icon: this.icon,
      icon_key: this.iconKey,
      desiredPosition: this.desiredPosition,
    };
  }
}

