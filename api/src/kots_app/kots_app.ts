import { Params } from "../server/params";
import { Stores } from "../schema/stores";
import zlib from "zlib";
import { KotsAppStore } from "./kots_app_store";
import { eq, eqIgnoringLeadingSlash, FilesAsBuffers, TarballUnpacker, isTgzByName } from "../troubleshoot/util";
import { kotsTemplateConfig, kotsEncryptString, kotsRewriteVersion } from "./kots_ffi";
import { ReplicatedError } from "../server/errors";
import { uploadUpdate } from "../controllers/kots/KotsAPI";
import { getS3 } from "../util/s3";
import tmp from "tmp";
import fs from "fs";
import path from "path";
import tar from "tar-stream";
import mkdirp from "mkdirp";
import { exec } from "child_process";
import { Cluster } from "../cluster";
import { putObject } from "../util/s3";
import * as _ from "lodash";
import yaml from "js-yaml";
import { ApplicationSpec } from "./kots_app_spec";
import { extractConfigValuesFromTarball } from "../util/tar";

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
  isConfigurable: boolean;
  snapshotTTL?: string;
  snapshotSchedule?: string;
  restoreInProgressName?: string;
  restoreUndeployStatus?: string;

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
  public async getKotsAppSpec(clusterId: string, kotsAppStore: KotsAppStore): Promise<ApplicationSpec | undefined> {
    const activeDownstream = await kotsAppStore.getCurrentVersion(this.id, clusterId);
    if (!activeDownstream) {
      return undefined;
    }

    return kotsAppStore.getKotsAppSpec(this.id, activeDownstream.parentSequence!);
  }
  public async getDownstreamGitOps(clusterId: string, stores: Stores): Promise<any> {
      const gitops = await stores.kotsAppStore.getDownstreamGitOps(this.id, clusterId);
      return gitops;
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

    const parsedKotsAppSpec = await stores.kotsAppStore.getKotsAppSpec(this.id, activeDownstream.parentSequence!);
    try {
      const parsedAppSpec = yaml.safeLoad(appSpec);
      const links: KotsAppLink[] = [];
      for (const unrealizedLink of parsedAppSpec.spec.descriptor.links) {
        // this is a pretty naive solution that works when there is 1 downstream only
        // we need to think about what the product experience is when
        // there are > 1 downstreams

        let rewrittenUrl = unrealizedLink.url;
        if (parsedKotsAppSpec && parsedKotsAppSpec.ports) {
          const mapped = _.find(parsedKotsAppSpec.ports, (port: any) => {
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
      JSON.parse(indexFiles.files[bundleIndexJsonPath].toString());

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

  private async getConfigDataFromFiles(files: FilesAsBuffers): Promise<ConfigData> {
    let configSpec: string = "",
        configValues: string = "",
        configValuesPath: string = "";

    for (const path in files.files) {
      try {
        const content = files.files[path];
        const parsedContent = yaml.safeLoad(content.toString());
        if (!parsedContent) {
          continue;
        }
        if (parsedContent.kind === "Config" && parsedContent.apiVersion === "kots.io/v1beta1") {
          configSpec = content.toString();
        } else if (parsedContent.kind === "ConfigValues" && parsedContent.apiVersion === "kots.io/v1beta1") {
          configValues = content.toString();
          configValuesPath = path;
        }
      } catch {
        // TODO: this will happen on multi-doc files.
      }
    }

    return {
      configSpec,
      configValues,
      configValuesPath,
    }
  }

  shouldUpdateConfigValues(configGroups: KotsConfigGroup[], configValues: any, item: KotsConfigItem): boolean {
    if (item.hidden || item.when === "false" || (item.type === "password" && item.value === this.getPasswordMask())) {
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

  async applyConfigValues(configSpec: string, configValues: string): Promise<KotsConfigGroup[]> {
    const templatedConfig = await kotsTemplateConfig(configSpec, configValues);

    if (!templatedConfig.spec || !templatedConfig.spec.groups) {
      throw new ReplicatedError("Config groups not found");
    }

    const parsedConfigValues = yaml.safeLoad(configValues);
    if (!parsedConfigValues.spec || !parsedConfigValues.spec.values) {
      throw new ReplicatedError("Config values not found");
    }

    const specConfigGroups = templatedConfig.spec.groups;
    const specConfigValues = parsedConfigValues.spec.values;

    specConfigGroups.forEach(group => {
      group.items.forEach(item => {
        if (item.type === "password") {
          if (item.value) {
            item.value = this.getPasswordMask();
          }
        } else if (item.name in specConfigValues) {
          item.value = specConfigValues[item.name].value;
        }
      });
    });

    return specConfigGroups;
  }

  async getAppConfigGroups(stores: Stores, appId: string, sequence: string): Promise<KotsConfigGroup[]> {
    try {
      const configData = await stores.kotsAppStore.getAppConfigData(appId, sequence);
      const { configSpec, configValues } = configData!;
      return await this.applyConfigValues(configSpec, configValues);
    } catch(err) {
      throw new ReplicatedError(`Failed to get config groups ${err}`);
    }
  }

  // TODO this function needs to be rewritten to simply use the archive and let kots do the work
  async updateAppConfig(stores: Stores, slug: string, sequence: string, updatedConfigGroups: KotsConfigGroup[], createNewVersion: boolean): Promise<void> {
    const tmpDir = tmp.dirSync();
    try {
      const paths: string[] = await this.getFilesPaths(sequence);
      const files: FilesAsBuffers = await this.getFiles(sequence, paths);

      const { configSpec, configValues } = await this.getConfigDataFromFiles(files);

      const parsedConfig = yaml.safeLoad(configSpec);
      const parsedConfigValues = yaml.safeLoad(configValues);

      const specConfigValues = parsedConfigValues.spec.values;
      const specConfigGroups = parsedConfig.spec.groups;

      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const encryptionKey = await stores.kotsAppStore.getAppEncryptionKey(appId, sequence);

      const downstreams = await stores.kotsAppStore.listDownstreamsForApp(appId);

      for (let g = 0; g < updatedConfigGroups.length; g++) {
        const group = updatedConfigGroups[g];
        for (let i = 0; i < group.items.length; i++) {
          const item = group.items[i];
          if (this.shouldUpdateConfigValues(specConfigGroups, specConfigValues, item)) {
            // these are "omitempty" in Go, but TS adds "null" strings in.
            let configVal = {};
            if (item.value) {
              let value = item.value;
              if (item.type === "password" && encryptionKey !== "") {
                value = await kotsEncryptString(encryptionKey, item.value);
              }
              configVal["value"] = value;
            }
            if (item.default) {
              configVal["default"] = item.default;
            }
            specConfigValues[item.name] = configVal;
          }
        }
      }

      const updatedConfigValues = yaml.safeDump(parsedConfigValues);

      // Get a fresh copy of the archive
      fs.writeFileSync(path.join(tmpDir.name, "input.tar.gz"), await this.getArchive(sequence));

      const inputArchive = path.join(tmpDir.name, "input.tar.gz");
      const outputArchive = path.join(tmpDir.name, "output.tar.gz");

      const app = await stores.kotsAppStore.getApp(appId);
      const registrySettings = await stores.kotsAppStore.getAppRegistryDetails(appId);
      await kotsRewriteVersion(app, inputArchive, downstreams, registrySettings, false, outputArchive, stores, updatedConfigValues);
      const outputTgzBuffer = fs.readFileSync(outputArchive);
      if (!createNewVersion) {
        const params = await Params.getParams();
        const objectStorePath = path.join(params.shipOutputBucket.trim(), appId, `${sequence}.tar.gz`);
        await putObject(params, objectStorePath, outputTgzBuffer, params.shipOutputBucket);
        const bufferConfigValues = await extractConfigValuesFromTarball(outputTgzBuffer);
        await stores.kotsAppStore.updateAppConfigValues(appId, sequence, bufferConfigValues!);
      } else {
        await uploadUpdate(stores, slug, outputTgzBuffer, "Config Change");
      }
    } finally {
      tmpDir.removeCallback();
    }
  }

  async templateConfigGroups(stores: Stores, appId: string, sequence: string, configGroups: KotsConfigGroup[]): Promise<KotsConfigGroup[]> {
    const configData = await stores.kotsAppStore.getAppConfigData(appId, sequence);
    const { configSpec, configValues } = configData!;

    const parsedConfig = yaml.safeLoad(configSpec);
    const parsedConfigValues = yaml.safeLoad(configValues);

    const specConfigValues = parsedConfigValues.spec.values;
    const specConfigGroups = parsedConfig.spec.groups;

    configGroups.forEach(group => {
      group.items.forEach(async item => {
        if (this.shouldUpdateConfigValues(specConfigGroups, specConfigValues, item)) {
          let configVal = {}
          if (item.value) {
            configVal["value"] = item.value;
          }
          if (item.default) {
            configVal["default"] = item.default;
          }
          specConfigValues[item.name] = configVal;
        }
      });
    });

    const updatedConfigValues = yaml.safeDump(parsedConfigValues);
    return await this.applyConfigValues(configSpec, updatedConfigValues);
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

  async getFiles(sequence: string, fileNames: string[]): Promise<FilesAsBuffers> {
    const fileNameList = fileNames.map((fileName) => ({
      path: fileName,
      matcher: eqIgnoringLeadingSlash(fileName),
    }));
    const filesWeWant = await this.downloadFiles(this.id, sequence, fileNameList);
    return filesWeWant;
  }

  async getFilesJSON(sequence: string, fileNames: string[]): Promise<string> {
    const files: FilesAsBuffers = await this.getFiles(sequence, fileNames);
    let fileStrings: {
      [key: string]: string;
    } = {};
    for (const path in files.files) {
      const content = files.files[path];
      if (isTgzByName(path)) {
        fileStrings[path] = content.toString('base64');
      } else {
        fileStrings[path] = content.toString();
      }
    }
    const jsonFiles = JSON.stringify(fileStrings);
    return jsonFiles;
  }

  async downloadFiles(appId: string, sequence: string, filesWeCareAbout: Array<{ path: string; matcher }>): Promise<FilesAsBuffers> {
    const replicatedParams = await Params.getParams();

    return new Promise<FilesAsBuffers>((resolve, reject) => {
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

  async render(sequence: string, overlayPath: string, kustomizeVersion: string|undefined): Promise<string> {
    const replicatedParams = await Params.getParams();
    const params = {
      Bucket: replicatedParams.shipOutputBucket,
      Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}${this.id}/${sequence}.tar.gz`,
    };

    const tgzStream = getS3(replicatedParams).getObject(params).createReadStream();
    const extract = tar.extract();
    const gzunipStream = zlib.createGunzip();

    return new Promise((resolve, reject) => {
      const tmpDir = tmp.dirSync();
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
        // Choose kustomize binary
        let kustomizeString = "kustomize3.5.4";
        if (kustomizeVersion && kustomizeVersion !== "") {
          if (kustomizeVersion !== "latest") {
            kustomizeString = `kustomize${kustomizeVersion}`;
          }
        }
        // Run kustomize
        exec(`${kustomizeString} build ${path.join(tmpDir.name, overlayPath)}`, { maxBuffer: 1024 * 5000 }, (err, stdout, stderr) => {
          tmpDir.removeCallback();
          if (err) {
            // logger.error({ msg: "err running kustomize", err, stderr })
            reject(err);
            return;
          }

          resolve(stdout);
        });
      });

      tgzStream.pipe(gzunipStream).pipe(extract);
    });
  }

  public async isGitOpsSupported(stores: Stores): Promise<boolean> {
    const sequence = this.currentSequence || 0;
    return await stores.kotsAppStore.isGitOpsSupported(this.id, sequence);
  }

  private async isAllowRollback(stores: Stores): Promise<boolean> {
    const parsedKotsAppSpec = await stores.kotsAppStore.getKotsAppSpec(this.id, this.currentSequence!);
    try {
      if (parsedKotsAppSpec && parsedKotsAppSpec.allowRollback) {
        return true;
      }
    } catch {
      /* not a valid app spec */
    }

    return false;
  }

  private async isAllowSnapshots(stores: Stores): Promise<boolean> {
    const parsedKotsAppSpec = await stores.kotsAppStore.getKotsAppSpec(this.id, this.currentSequence!);
    const partOfLicenseYaml = await stores.kotsAppStore.isAllowSnapshotsPartOfLicenseYaml(this.id, this.currentSequence!);

    try {
      if (parsedKotsAppSpec && parsedKotsAppSpec.allowSnapshots && partOfLicenseYaml) {
        return true;
      }
    } catch {
      /* not a valid app spec */
    }
    return false;
  }

  private async getKotsLicenseType(stores: Stores): Promise<string> {
    const id = await stores.kotsAppStore.getIdFromSlug(this.slug);
    const sequence = await stores.kotsAppStore.getMaxSequence(id);
    return await stores.kotsAppStore.getKotsAppLicenseType(id, sequence);
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
      curl https://krew.sh/support-bundle | bash
      kubectl support-bundle ${params.apiAdvertiseEndpoint}/api/v1/troubleshoot/${watchSlug}
    `;
    return bundleCommand;
  }


  public toSchema(downstreams: Cluster[], stores: Stores) {
    return {
      ...this,
      isGitOpsSupported: () => this.isGitOpsSupported(stores),
      allowRollback: () => this.isAllowRollback(stores),
      allowSnapshots: () => this.isAllowSnapshots(stores),
      currentVersion: () => this.getCurrentAppVersion(stores),
      licenseType: () => this.getKotsLicenseType(stores),
      downstreams: _.map(downstreams, (downstream) => {
        const kotsSchemaCluster = downstream.toKotsAppSchema(this.id, stores);
        return {
          name: downstream.title,
          gitops: () => this.getDownstreamGitOps(downstream.id, stores),
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
  commitUrl?: string;
  gitDeployable?: boolean;
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
  registryPasswordEnc: string;
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
  help_text: string;
  recommended: boolean;
  default: string;
  value: string;
  data: string;
  multi_value: [string];
  readonly: boolean;
  write_once: boolean;
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
  renderError: string | null;
}

export interface ConfigData {
  configSpec: string;
  configValues: string;
  configValuesPath?: string;
}
