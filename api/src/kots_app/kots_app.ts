import { Params } from "../server/params";
import { Stores } from "../schema/stores";
import zlib from "zlib";
import { KotsAppStore } from "./kots_app_store";
import { eq, eqIgnoringLeadingSlash, FilesAsString, TarballUnpacker, TarballPacker } from "../troubleshoot/util";
import { ReplicatedError } from "../server/errors";
import { uploadUpdate } from "../controllers/kots/KotsAPI";
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
import yaml from "js-yaml";

export class KotsApp {
  id: string;
  name: string;
  license?: string;
  iconUri: string;
  upstreamUri: string;
  createdAt: Date;
  updatedAt?: Date;
  slug: string;
  currentSequence?: number;
  lastUpdateCheckAt?: Date;
  bundleCommand: string;
  currentVersion: KotsVersion;
  airgapUploadPending: boolean;
  isAirgap: boolean;
  hasPreflight: boolean;

  // Version Methods
  public async getCurrentAppVersion(stores: Stores): Promise<KotsVersion | undefined> {
    // this is to get the current version of the upsteam from the app_version table
    // annoying to have a separate method for this but the others require a clusteId.
    // good candidate for a refactor
    return stores.kotsAppStore.getCurrentAppVersion(this.id);
  }
  public async getCurrentVersion(clusterId: string, stores: Stores): Promise<KotsVersion | undefined> {
    return stores.kotsAppStore.getCurrentVersion(this.id, clusterId);
  }
  public async getPendingVersions(clusterId: string, stores: Stores): Promise<KotsVersion[]> {
    return stores.kotsAppStore.listPendingVersions(this.id, clusterId);
  }
  public async getPastVersions(clusterId: string, stores: Stores): Promise<KotsVersion[]> {
    return stores.kotsAppStore.listPastVersions(this.id, clusterId);
  }
  public async getKotsAppSpec(clusterId: string, kotsAppStore: KotsAppStore): Promise<any> {
    const activeDownstream = await kotsAppStore.getCurrentVersion(this.id, clusterId);
    if (!activeDownstream) {
      return null;
    }

    const kotsAppSpec = await kotsAppStore.getKotsAppSpec(this.id, activeDownstream.parentSequence!);
    if (!kotsAppSpec) {
      return null;
    }
    return yaml.safeLoad(kotsAppSpec);
  }
  public async getRealizedLinksFromAppSpec(clusterId: string, stores: Stores): Promise<KotsAppLink[]> {
    const activeDownstream = await stores.kotsAppStore.getCurrentVersion(this.id, clusterId);
    if (!activeDownstream) {
      return [];
    }

    const appSpec = await stores.kotsAppStore.getAppSpec(this.id, activeDownstream.parentSequence!);
    if (!appSpec) {
      return [];
    }

    const kotsAppSpec = await stores.kotsAppStore.getKotsAppSpec(this.id, activeDownstream.parentSequence!);
    try {
      const parsedAppSpec = yaml.safeLoad(appSpec);
      const parsedKotsAppSpec = kotsAppSpec ? yaml.safeLoad(kotsAppSpec) : null;
      const links: KotsAppLink[] = [];
      for (const unrealizedLink of parsedAppSpec.spec.descriptor.links) {
        // this is a pretty naive solution that works when there is 1 downstream only
        // we need to think about what the product experience is when
        // there are > 1 downstreams

        let rewrittenUrl = unrealizedLink.url;
        if (parsedKotsAppSpec) {
          const mapped = _.find(parsedKotsAppSpec.spec.ports, (port: any) => {
            return port.applicationUrl === unrealizedLink.url;
          });
          if (mapped) {
            rewrittenUrl = parsedAppSpec ? `http://localhost:${mapped.localPort}`: unrealizedLink;
          }
        }

        const realized: KotsAppLink = {
          title: unrealizedLink.description,
          uri: rewrittenUrl,
        };

        links.push(realized);
      }

      return links;
    } catch (err) {
      console.log(err);
      return [];
    }
  }

  async getFilesPaths(sequence: string): Promise<string[]> {
    const bundleIndexJsonPath = "index.json";
    const indexFiles = await this.downloadFiles(this.id, sequence, [{
      path: bundleIndexJsonPath,
      matcher: eq(bundleIndexJsonPath),
    }]);

    const index = indexFiles.files[bundleIndexJsonPath] &&
      JSON.parse(indexFiles.files[bundleIndexJsonPath]);

    let paths: string[] = [];
    if (!index) {
      paths = indexFiles.fakeIndex;
    } else {
      index.map((p) => (paths.push(p.path)));
    }

    return paths;
  }

  getPasswordMask(): string {
    return "***HIDDEN***";
  }

  getOriginalItem(groups: KotsConfigGroup[], itemName: string) {
    for (let g = 0; g < groups.length; g++) {
      const group = groups[g];
      for (let i = 0; i < group.items.length; i++) {
        const item = group.items[i];
        if (item.name === itemName) {
          return item;
        }
      }
    }
    return null;
  }

  async getConfigData(files: FilesAsString): Promise<any> {
    let configContent: any = {},
        configPath: string = "",
        configValuesContent: any = {},
        configValuesPath: string = "";

    for (const path in files.files) {
      try {
        const content = yaml.safeLoad(files.files[path]);
        if (!content) {
          continue;
        }
        if (content.kind === "Config" && content.apiVersion === "kots.io/v1beta1") {
          configContent = content;
          configPath = path;
        } else if (content.kind === "ConfigValues" && content.apiVersion === "kots.io/v1beta1") {
          configValuesContent = content;
          configValuesPath = path;
        }
      } catch {
        // this will happen on multi-doc files.
      }
    }

    return {
      configContent,
      configPath,
      configValuesContent,
      configValuesPath,
    }
  }

  shouldUpdateConfigValues(configGroups: KotsConfigGroup[], configValues: any, item: KotsConfigItem): boolean {
    if (item.hidden || (item.type === "password" && item.value === this.getPasswordMask())) {
      return false;
    }
    if (item.name in configValues) {
      return item.value !== configValues[item.name];
    } else {
      const originalItem = this.getOriginalItem(configGroups, item.name);
      if (originalItem && item.value) {
        if (originalItem.value) {
          return item.value !== originalItem.value;
        } else if (originalItem.default) {
          return item.value !== originalItem.default;
        } else {
          return true;
        }
      }
    }
    return false;
  }

  async getConfigGroups(sequence: string): Promise<KotsConfigGroup[]> {
    const paths: string[] = await this.getFilesPaths(sequence);
    const files: FilesAsString = await this.getFiles(sequence, paths);

    const { configContent, configValuesContent } = await this.getConfigData(files);

    let configValues = {}, configGroups: KotsConfigGroup[] = [];
    if (configContent.spec && configContent.spec.groups) {
      configGroups = configContent.spec.groups;
    }
    if (configValuesContent.spec && configValuesContent.spec.values) {
      configValues = configValuesContent.spec.values;
    }

    configGroups.forEach(group => {
      group.items.forEach(item => {
        if (item.type === "password") {
          item.value = this.getPasswordMask();
        } else if (item.name in configValues) {
          item.value = configValues[item.name];
        }
      });
    });
    return configGroups;
  }

  async updateAppConfig(stores: Stores, slug: string, sequence: string, updatedConfigGroups: KotsConfigGroup[]): Promise<void> {
    const paths: string[] = await this.getFilesPaths(sequence);
    const files: FilesAsString = await this.getFiles(sequence, paths);

    const { configContent, configValuesContent, configValuesPath } = await this.getConfigData(files);

    let configGroups: KotsConfigGroup[] = [];
    if (configContent.spec && configContent.spec.groups) {
      configGroups = configContent.spec.groups;
    }

    if (!configValuesContent.spec || !configValuesContent.spec.values || configValuesPath === "") {
      throw new ReplicatedError("No config values were found in the files list");
    }

    const configValues = configValuesContent.spec.values;
    updatedConfigGroups.forEach(group => {
      group.items.forEach(async item => {
        if (this.shouldUpdateConfigValues(configGroups, configValues, item)) {
          configValues[item.name] = item.value;
        }
      });
    });

    files.files[configValuesPath] = yaml.safeDump(configValuesContent);

    const bundlePacker = new TarballPacker();
    const tarGzBuffer: Buffer = await bundlePacker.packFiles(files);

    await uploadUpdate(stores, slug, tarGzBuffer, "Config Change");
  }

  // Source files
  async generateFileTreeIndex(sequence) {
    const paths = await this.getFilesPaths(sequence);
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
              logger.error({ msg: "err running kustomize", err, stderr })
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

  private async isAppConfigurable(): Promise<boolean> {
    const sequence = Number.isInteger(this.currentSequence!) ? `${this.currentSequence}` : "";
    if (sequence === "") {
      return false;
    }
    const paths: string[] = await this.getFilesPaths(sequence);
    const files: FilesAsString = await this.getFiles(sequence, paths);
    const { configPath } = await this.getConfigData(files);
    return configPath !== "";
  }

  private async isAllowRollback(stores: Stores): Promise<boolean> {
    const kotsAppSpec = await stores.kotsAppStore.getKotsAppSpec(this.id, this.currentSequence!);
    try {
      const parsedKotsAppSpec = kotsAppSpec ? yaml.safeLoad(kotsAppSpec) : null;
      if (parsedKotsAppSpec && parsedKotsAppSpec.spec.allowRollback) {
        return true;
      }
    } catch {
      /* not a valid app spec */
    }

    return false;
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

  public async getSupportBundleCommand(watchSlug: string): Promise<string> {
    const params = await Params.getParams();
    const bundleCommand = `
      kubectl krew install support-bundle
      kubectl support-bundle ${params.apiAdvertiseEndpoint}/api/v1/troubleshoot/${watchSlug}
    `;
    return bundleCommand;
  }


  public toSchema(downstreams: Cluster[], stores: Stores) {
    return {
      ...this,
      isConfigurable: () => this.isAppConfigurable(),
      allowRollback: () => this.isAllowRollback(stores),
      currentVersion: () => this.getCurrentAppVersion(stores),
      downstreams: _.map(downstreams, (downstream) => {
        const kotsSchemaCluster = downstream.toKotsAppSchema(this.id, stores);
        return {
          name: downstream.title,
          links: () => this.getRealizedLinksFromAppSpec(kotsSchemaCluster.id, stores),
          currentVersion: () => this.getCurrentVersion(downstream.id, stores),
          pastVersions: () => this.getPastVersions(downstream.id, stores),
          pendingVersions: () => this.getPendingVersions(downstream.id, stores),
          cluster: kotsSchemaCluster
        };
      }),
    };
  }
}

export interface KotsAppLink {
  title: string;
  uri: string;
}

export interface KotsVersion {
  title: string;
  status: string;
  createdOn: string;
  parentSequence?: number;
  sequence: number;
  releaseNotes: string;
  deployedAt: string;
  preflightResult: string;
  preflightResultCreatedAt: string;
  hasError?: boolean;
  source?: string;
  diffSummary?: string;
}

export interface KotsAppMetadata {
  name: string;
  iconUri: string;
  namespace: string;
  isKurlEnabled: boolean;
}

export interface AppRegistryDetails {
  appSlug: string;
  hostname: string;
  username: string;
  password: string;
  namespace: string;
}

export interface KotsAppRegistryDetails {
  registryHostname: string;
  registryUsername: string;
  registryPassword: string;
  namespace: string;
  lastSyncedAt: string;
}

export interface KotsConfigChildItem {
  name: string;
  title: string;
  recommended: boolean;
  default: string;
  value: string;
}

export interface KotsConfigItem {
  name: string;
  type: string;
  title: string;
  helpText: string;
  recommended: boolean;
  default: string;
  value: string;
  multiValue: [string];
  readOnly: boolean;
  writeOnce: boolean;
  when: string;
  multiple: boolean;
  hidden: boolean;
  position: number;
  affix: string;
  required: boolean;
  items: KotsConfigChildItem[];
}

export interface KotsConfigGroup {
  name: string;
  title: string;
  description: string;
  items: KotsConfigItem[];
}

export interface KotsDownstreamOutput {
  dryrunStdout: string;
  dryrunStderr: string;
  applyStdout: string;
  applyStderr: string;
}
