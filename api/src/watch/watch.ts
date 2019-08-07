import { Cluster } from "../cluster/cluster";
import { Feature } from "../feature/feature";
import { Stores } from "../schema/stores";
import { NotificationQueries } from "../notification";
import { TroubleshootQueries } from "../troubleshoot";
import zlib from "zlib";
import { eq, eqIgnoringLeadingSlash, FilesAsString, TarballUnpacker } from "../troubleshoot/util";
import { getS3 } from "../util/s3";
import { Params } from "../server/params";
import { Context } from "../context";
import { Entitlement } from '../license';
import _ from "lodash";
import yaml from "js-yaml";
import { logger } from "../server/logger";

export class Watch {
  public id: string;
  public stateJSON: string;
  public watchName: string;
  public slug: string;
  public watchIcon: string;
  public lastUpdated: string;
  public createdOn: string;
  public contributors: [Contributor];
  public notifications: [Notification];
  public features: [Feature];
  public cluster: Cluster;
  public watches: [Watch];
  public currentVersion: Version;
  public pendingVersions: [Version];
  public pastVersions: [Version];
  public parentWatch: Watch;
  public metadata: string;
  public config?: Array<ConfigGroup>;
  public entitlements?: Array<Entitlement>;
  public lastUpdateCheck: string;
  public bundleCommand?: string;
  public hasPreflight: boolean;

  // Watch Cluster Methods
  public async getCluster(stores: Stores): Promise<Cluster | void> {
    return stores.clusterStore.getForWatch(this.id)
  }

  // Parent/Child Watch Methods
  public async getParentWatch(stores: Stores): Promise<Watch> {
    const parentWatchId = await stores.watchStore.getParentWatchId(this.id)
    return stores.watchStore.getWatch(parentWatchId);
  }
  public async getChildWatches(stores: Stores): Promise<Watch[]> {
    return stores.watchStore.listWatches(undefined, this.id);
  }

  // Version Methods
  public async getCurrentVersion(stores: Stores): Promise<Version | undefined> {
    return stores.watchStore.getCurrentVersion(this.id);
  }
  public async getPendingVersions(stores: Stores): Promise<Version[]> {
    return stores.watchStore.listPendingVersions(this.id);
  }
  public async getPastVersions(stores: Stores): Promise<Version[]> {
    return stores.watchStore.listPastVersions(this.id);
  }

  // Contributor Methods
  public async getContributors(stores: Stores): Promise<Contributor[]> {
    return stores.watchStore.listWatchContributors(this.id);
  }

  // Features Methods
  public async getFeatures(stores: Stores): Promise<Feature[]> {
    const features = await stores.featureStore.listWatchFeatures(this.id);
    const result = _.map(features, (feature: Feature) => {
      return {
        ...feature,
      };
    });
    return result;
  }

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

  generateConfigGroups(stateJSON: string): Array<ConfigGroup> {
    try {
      const doc = yaml.safeLoad(stateJSON);
      const config = doc.v1.config;
      const configSpecArr = yaml.safeLoad(doc.v1.upstreamContents.appRelease.configSpec);

      const configGroups: Array<ConfigGroup> = [];
      _.map(configSpecArr.v1, (configSpec: ConfigGroup) => {
        const filteredConfigSpec: ConfigGroup = { ...configSpec, items: [] };
        _.map(configSpec.items, (item: ConfigItem) => {
          if (item.name in config) {
            item.value = config[item.name]
            filteredConfigSpec.items.push(item);
          }
        });
        if (filteredConfigSpec.items.length) {
          configGroups.push(filteredConfigSpec);
        }
      })
      return configGroups;
    } catch (err) {
      return [];
    }
  }

  getEntitlementsWithNames(stores: Stores): Array<Entitlement> {
    try {
      const doc = yaml.safeLoad(this.stateJSON);
      const appRelease = doc.v1.upstreamContents.appRelease;

      if (!appRelease.entitlements.values) {
        return [];
      }

      const entitlements = appRelease.entitlements.values;
      const entitlementSpec = appRelease.entitlementSpec;

      return stores.licenseStore.getEntitlementsWithNames(entitlements, entitlementSpec);
    } catch (err) {
      return [];
    }
  }

  public toSchema(root: any, stores: Stores, context: Context): any {
    return {
      ...this,
      watches: async () => (await this.getChildWatches(stores)).map(watch => watch.toSchema(root, stores, context)),
      cluster: async () => await this.getCluster(stores),
      contributors: async () => this.getContributors(stores),
      notifications: async () => NotificationQueries(stores).listNotifications(root, { watchId: this.id }, context),
      features: async () => this.getFeatures(stores),
      pendingVersions: async () => this.getPendingVersions(stores),
      pastVersions: async () => this.getPastVersions(stores),
      currentVersion: async () => this.getCurrentVersion(stores),
      parentWatch: async () => this.getParentWatch(stores),
      config: async () => this.generateConfigGroups(this.stateJSON),
      entitlements: async () => this.getEntitlementsWithNames(stores),
      bundleCommand: async () => TroubleshootQueries(stores).getSupportBundleCommand(root, { watchSlug: this.slug }, context)
    };
  }
}

export interface Version {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
  deployedAt: string;
}

export interface VersionDetail {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
  rendered: string;
  deployedAt: string;
}

export interface StateMetadata {
  name: string;
  icon: string;
  version: string;
}

export interface Contributor {
  id: string;
  createdAt: string;
  githubId: number;
  login: string;
  avatar_url: string;
}

export interface ConfigItem {
  name: string;
  title: string;
  default: string;
  value: string;
  type: string;
}

export interface ConfigGroup {
  name: string;
  title: string;
  description: string;
  items: Array<ConfigItem>
}

export function parseWatchName(watchName: string): string {
  if (watchName.startsWith("replicated.app") || watchName.startsWith("staging.replicated.app") || watchName.startsWith("local.replicated.app")) {
    const splitReplicatedApp = watchName.split("/");
    if (splitReplicatedApp.length < 2) {
      return watchName;
    }

    const splitReplicatedAppParams = splitReplicatedApp[1].split("?");
    return splitReplicatedAppParams[0];
  }

  return watchName;
}
