import ffi from "ffi";
import Struct from "ref-struct";
import { Stores } from "../schema/stores";
import { KotsApp } from "./";
import { Params } from "../server/params";
import { putObject } from "../util/s3";
import path from "path";
import tmp from "tmp";
import fs from "fs";
import {
  extractDownstreamNamesFromTarball,
  extractInstallationSpecFromTarball,
  extractRawInstallationSpecFromTarball,
  extractPreflightSpecFromTarball,
  extractSupportBundleSpecFromTarball,
  extractAppSpecFromTarball,
  extractKotsAppSpecFromTarball,
  extractAppTitleFromTarball,
  extractAppIconFromTarball,
  extractKotsAppLicenseFromTarball,
  extractAnalyzerSpecFromTarball,
  extractConfigSpecFromTarball,
  extractConfigValuesFromTarball,
  extractBackupSpecFromTarball,
} from "../util/tar";
import { KotsAppRegistryDetails } from "../kots_app"
import { Cluster } from "../cluster";
import * as _ from "lodash";
import yaml from "js-yaml";
import { StatusServer } from "../airgap/status";
import { getDiffSummary } from "../util/utilities";
import { ReplicatedError } from "../server/errors";
import { createGitCommitForVersion } from "./gitops";

const GoString = Struct({
  p: "string",
  n: "longlong"
});

const GoBool = "bool";

function kots() {
  return ffi.Library("/lib/kots.so", {
    TestRegistryCredentials: ["void", [GoString, GoString, GoString, GoString, GoString]],
    UpdateCheck: ["void", [GoString, GoString, GoString]],
    UpdateDownload: ["void", [GoString, GoString, GoString, GoString, GoString]],
    UpdateDownloadFromAirgap: ["void", [GoString, GoString, GoString, GoString, GoString]],
    RewriteVersion: ["void", [GoString, GoString, GoString, GoString, GoString, GoString, GoBool, GoBool, GoString]],
    TemplateConfig: [GoString, [GoString, GoString, GoString, GoString]],
    EncryptString: [GoString, [GoString, GoString]],
    DecryptString: [GoString, [GoString, GoString]],
    RenderFile: ["void", [GoString, GoString, GoString, GoString]],
  });
}

export interface Update {
  cursor: string;
  versionLabel: string;
}

export async function kotsAppDownloadUpdates(updatesAvailable: Update[], app: KotsApp, stores: Stores): Promise<void> {
  const registryInfo = await stores.kotsAppStore.getAppRegistryDetails(app.id);
  for (let i = 0; i < updatesAvailable.length; i++) {
    const update = updatesAvailable[i];
    try {
      await stores.kotsAppStore.setUpdateDownloadStatus(`Downloading release ${update.versionLabel}`, "running");
      await kotsAppDownloadUpdate(update.cursor, app, registryInfo, stores);
    } catch (err) {
      if (i === updatesAvailable.length - 1) {
        console.error(`Failed to download release ${update.cursor}: ${err}`);
        throw err;
      }
    }
  }
}

export async function kotsAppDownloadUpdate(cursor: string, app: KotsApp, registryInfo: KotsAppRegistryDetails, stores: Stores): Promise<boolean> {
  // We need to include the last archive because if there is an update, the ffi function will update it
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive("" + (app.currentSequence!)));
    const namespace = getK8sNamespace();

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    const namespaceParam = new GoString();
    namespaceParam["p"] = namespace;
    namespaceParam["n"] = namespace.length;

    const cursorParam = new GoString();
    cursorParam["p"] = cursor;
    cursorParam["n"] = cursor.length;

    const registryJson = JSON.stringify(registryInfo)
    const registryJsonParam = new GoString();
    registryJsonParam["p"] = registryJson;
    registryJsonParam["n"] = registryJson.length;

    kots().UpdateDownload(socketParam, archiveParam, namespaceParam, registryJsonParam, cursorParam);
    await statusServer.connection();
    const isUpdateAvailable: number = await statusServer.termination((resolve, reject, obj): boolean => {
      if (obj.status === "running") {
        stores.kotsAppStore.setUpdateDownloadStatus(obj.display_message, "running");
        return false;
      }
      if (obj.status === "terminated") {
        if (obj.exit_code !== -1) {
          resolve(obj.exit_code);
        } else {
          reject(new Error(obj.display_message));
        }
        return true;
      }
      return false;
    });

    if (isUpdateAvailable < 0) {
      console.log("error downloading update")
      return false;
    }

    if (isUpdateAvailable > 0) {
      await saveUpdateVersion(archive, app, stores, "Upstream Update");
    }

    return isUpdateAvailable > 0;
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsAppDownloadUpdateFromAirgap(airgapFile: string, app: KotsApp, registryInfo: KotsAppRegistryDetails, stores: Stores): Promise<void> {
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive("" + (app.currentSequence!)));
    const namespace = getK8sNamespace();

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    const namespaceParam = new GoString();
    namespaceParam["p"] = namespace;
    namespaceParam["n"] = namespace.length;

    const airgapFileParam = new GoString();
    airgapFileParam["p"] = airgapFile;
    airgapFileParam["n"] = airgapFile.length;

    const registryJson = JSON.stringify(registryInfo)
    const registryJsonParam = new GoString();
    registryJsonParam["p"] = registryJson;
    registryJsonParam["n"] = registryJson.length;

    kots().UpdateDownloadFromAirgap(socketParam, archiveParam, namespaceParam, registryJsonParam, airgapFileParam);
    await statusServer.connection();
    const isUpdateAvailable: number = await statusServer.termination((resolve, reject, obj): boolean => {
      if (obj.status === "running") {
        stores.kotsAppStore.setUpdateDownloadStatus(obj.display_message, "running");
        return false;
      }
      if (obj.status === "terminated") {
        if (obj.exit_code !== -1) {
          resolve(obj.exit_code);
        } else {
          reject(new Error(obj.display_message));
        }
        return true;
      }
      return false;
    });

    if (isUpdateAvailable) {
      await saveUpdateVersion(archive, app, stores, "Airgap Upload");
    }

  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsRenderFile(app: KotsApp, stores: Stores, input: string, registryInfo: KotsAppRegistryDetails): Promise<string> {
  const filename = tmp.tmpNameSync();
  fs.writeFileSync(filename, input);

  // for the status server
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive("" + (app.currentSequence!)));

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const filepathParam = new GoString();
    filepathParam["p"] = filename;
    filepathParam["n"] = filename.length;

    const archivePathParam = new GoString();
    archivePathParam["p"] = archive;
    archivePathParam["n"] = archive.length;

    const registryJson = JSON.stringify(registryInfo)
    const registryJsonParam = new GoString();
    registryJsonParam["p"] = registryJson;
    registryJsonParam["n"] = registryJson.length;

    kots().RenderFile(socketParam, filepathParam, archivePathParam, registryJsonParam);
    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        if (obj.exit_code !== -1) {
          resolve();
        } else {
          reject(new Error(obj.display_message));
        }
        return true;
      }
      return false;
    });

    const rendered = fs.readFileSync(filename);
    return rendered.toString();
  } finally {
    tmpDir.removeCallback();
  }
}

async function saveUpdateVersion(archive: string, app: KotsApp, stores: Stores, updateSource: string) {
  // if there was an update available, expect that the new archive is in the smae place as the one we pased in
  const params = await Params.getParams();
  const buffer = fs.readFileSync(archive);
  const newSequence = (await stores.kotsAppStore.getMaxSequence(app.id)) + 1;
  const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, `${newSequence}.tar.gz`);
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const rawInstallationSpec = await extractRawInstallationSpecFromTarball(buffer);
  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);
  const configSpec = await extractConfigSpecFromTarball(buffer);
  const configValues = await extractConfigValuesFromTarball(buffer);
  const backupSpec = await extractBackupSpecFromTarball(buffer);

  console.log(`Save new version ${app.id}:${newSequence}, cursor=${installationSpec.cursor}`);

  await stores.kotsAppStore.createMidstreamVersion(
    app.id,
    newSequence,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.channelName,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    rawInstallationSpec,
    kotsAppLicense,
    configSpec,
    configValues,
    appTitle,
    appIcon,
    backupSpec
  );

  const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(app.id);
  for (const clusterId of clusterIds) {
    const downstreamGitops = await stores.kotsAppStore.getDownstreamGitOps(app.id, clusterId);

    let commitUrl = "";
    let gitDeployable = false;
    if (downstreamGitops.enabled) {
      const commitMessage = `Updates to the upstream of ${app.name}`;
      commitUrl = await createGitCommitForVersion(stores, app.id, clusterId, newSequence, commitMessage);
      if (commitUrl !== "") {
        gitDeployable = true;
      }
    }

    const status = preflightSpec
      ? "pending_preflight"
      : "pending";
    const diffSummary = await getDiffSummary(app);
    await stores.kotsAppStore.createDownstreamVersion(app.id, newSequence, clusterId, installationSpec.versionLabel, status, updateSource, diffSummary, commitUrl, gitDeployable);
  }
}

export async function kotsPullFromLicense(socket: string, out: string, kotsApp: KotsApp, downstreamName: string, stores: Stores) {
  const namespace = getK8sNamespace();

  const socketParam = new GoString();
  socketParam["p"] = socket;
  socketParam["n"] = socket.length;

  const licenseDataParam = new GoString();
  licenseDataParam["p"] = kotsApp.license;
  licenseDataParam["n"] = Buffer.from(kotsApp.license!).length;

  const downstreamParam = new GoString();
  downstreamParam["p"] = downstreamName;
  downstreamParam["n"] = downstreamName.length;

  const namespaceParam = new GoString();
  namespaceParam["p"] = namespace;
  namespaceParam["n"] = namespace.length;

  const outParam = new GoString();
  outParam["p"] = out;
  outParam["n"] = out.length;

  kots().PullFromLicense(socketParam, licenseDataParam, downstreamParam, namespaceParam, outParam);

  // args are returned so they are not garbage collected before native code is done
  return {
    socketParam,
    licenseDataParam,
    downstreamParam,
    namespaceParam,
    outParam,
  };
}

export async function kotsAppFromData(out: string, kotsApp: KotsApp, stores: Stores): Promise<{ hasPreflight: boolean, isConfigurable: boolean }> {
  const params = await Params.getParams();
  const buffer = fs.readFileSync(out);

  const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const rawInstallationSpec = await extractRawInstallationSpecFromTarball(buffer);
  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);
  const configSpec = await extractConfigSpecFromTarball(buffer);
  const configValues = await extractConfigValuesFromTarball(buffer);
  kotsApp.isConfigurable = !!configSpec;
  kotsApp.hasPreflight = !!preflightSpec;
  const backupSpec = await extractBackupSpecFromTarball(buffer);

  if (kotsAppLicense) {
    // update kots app with latest license
    await stores.kotsAppStore.updateKotsAppLicense(kotsApp.id, kotsAppLicense);
  }

  await stores.kotsAppStore.createMidstreamVersion(
    kotsApp.id,
    0,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.channelName,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    rawInstallationSpec,
    kotsAppLicense,
    configSpec,
    configValues,
    appTitle,
    appIcon,
    backupSpec
  );

  const downstreams = await extractDownstreamNamesFromTarball(buffer);
  const clusters = await stores.clusterStore.listAllUsersClusters();

  for (const downstream of downstreams) {
    const cluster = _.find(clusters, (c: Cluster) => {
      return c.title === downstream;
    });

    if (!cluster) {
      continue;
    }

    const downstreamState = kotsApp.isConfigurable
      ? "pending_config"
      : kotsApp.hasPreflight
        ? "pending_preflight"
        : "deployed";

    await stores.kotsAppStore.createDownstream(kotsApp.id, downstream, cluster.id);
    await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, installationSpec.versionLabel, downstreamState, "Kots Install", "", "", false);
  }

  return {
    isConfigurable: kotsApp.isConfigurable,
    hasPreflight: kotsApp.hasPreflight,
  };
}

export async function kotsAppFromAirgapData(out: string, app: KotsApp, stores: Stores): Promise<{ hasPreflight: boolean, isConfigurable: boolean }> {
  const params = await Params.getParams();
  const buffer = fs.readFileSync(out);
  const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, "0.tar.gz");
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const rawInstallationSpec = await extractRawInstallationSpecFromTarball(buffer);
  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);
  const configSpec = await extractConfigSpecFromTarball(buffer);
  const configValues = await extractConfigValuesFromTarball(buffer);
  const backupSpec = await extractBackupSpecFromTarball(buffer);

  await stores.kotsAppStore.createMidstreamVersion(
    app.id,
    0,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.channelName,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    rawInstallationSpec,
    kotsAppLicense,
    configSpec,
    configValues,
    appTitle,
    appIcon,
    backupSpec
  );

  const downstreams = await extractDownstreamNamesFromTarball(buffer);
  const clusters = await stores.clusterStore.listAllUsersClusters();
  for (const downstream of downstreams) {
    const cluster = _.find(clusters, (c: Cluster) => {
      return c.title === downstream;
    });

    if (!cluster) {
      continue;
    }

    await stores.kotsAppStore.createDownstream(app.id, downstream, cluster.id);
    await stores.kotsAppStore.createDownstreamVersion(app.id, 0, cluster.id, installationSpec.versionLabel, "deployed", "Airgap Upload", "", "", false);
  }

  await stores.kotsAppStore.setKotsAirgapAppInstalled(app.id);

  return {
    hasPreflight: !!preflightSpec,
    isConfigurable: !!configSpec,
  };
}

export async function kotsTestRegistryCredentials(endpoint: string, username: string, password: string, repo: string): Promise<String> {
  const tmpDir = tmp.dirSync();
  try {
    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const endpointParam = new GoString();
    endpointParam["p"] = endpoint;
    endpointParam["n"] = endpoint.length;

    const usernameParam = new GoString();
    usernameParam["p"] = username;
    usernameParam["n"] = username.length;

    const passwordParam = new GoString();
    passwordParam["p"] = password;
    passwordParam["n"] = password.length;

    const repoParam = new GoString();
    repoParam["p"] = repo;
    repoParam["n"] = repo.length;

    kots().TestRegistryCredentials(socketParam, endpointParam, usernameParam, passwordParam, repoParam);

    let testError = "";
    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        if (obj.exit_code !== 0) {
          testError = obj.display_message;
        }
        resolve();
        return true;
      }
      return false;
    });

    return testError;

  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsTemplateConfig(configSpec: string, configValues: string, license: string, registryInfo: KotsAppRegistryDetails): Promise<any> {
  const configDataParam = new GoString();
  configDataParam["p"] = configSpec;
  configDataParam["n"] = Buffer.from(configSpec).length;

  const configValuesDataParam = new GoString();
  configValuesDataParam["p"] = configValues;
  configValuesDataParam["n"] = Buffer.from(configValues).length;

  const licenseParam = new GoString();
  licenseParam["p"] = license;
  licenseParam["n"] = Buffer.from(license).length;

  const registryJson = JSON.stringify(registryInfo)
  const registryJsonParam = new GoString();
  registryJsonParam["p"] = registryJson;
  registryJsonParam["n"] = registryJson.length;

  const templatedConfig = kots().TemplateConfig(configDataParam, configValuesDataParam, licenseParam, registryJsonParam);
  if (templatedConfig == "" || templatedConfig["p"] == "") {
    throw new ReplicatedError("failed to template config");
  }

  try {
    return yaml.safeLoad(templatedConfig["p"]);
  } catch (err) {
    throw new ReplicatedError(`Failed to parse templated config ${err}`);
  }
}

export async function kotsEncryptString(cipherString: string, message: string): Promise<string> {
  const cipherStringParam = new GoString();
  cipherStringParam["p"] = cipherString;
  cipherStringParam["n"] = Buffer.from(cipherString).length;

  const messageParam = new GoString();
  messageParam["p"] = message;
  messageParam["n"] = Buffer.from(message).length;

  const encrypted = kots().EncryptString(cipherStringParam, messageParam);

  if (encrypted["p"] === null) {
    throw new ReplicatedError("Failed to encrypt string via FFI call");
  }

  return encrypted["p"];
}

export async function kotsDecryptString(cipherString: string, message: string): Promise<string> {
  const cipherStringParam = new GoString();
  cipherStringParam["p"] = cipherString;
  cipherStringParam["n"] = Buffer.from(cipherString).length;

  const messageParam = new GoString();
  messageParam["p"] = message;
  messageParam["n"] = Buffer.from(message).length;

  const decrypted = kots().DecryptString(cipherStringParam, messageParam);

  if (decrypted["p"] === null) {
    throw new ReplicatedError("Failed to encrypt string via FFI call");
  }

  return decrypted["p"];
}

export async function kotsRewriteVersion(app: KotsApp, archive: string, downstreams: string[], registryInfo: KotsAppRegistryDetails, copyImages: boolean, outputFile: string, stores: Stores, updatedConfigValues: string): Promise<string> {
  const tmpDir = tmp.dirSync();
  try {
    const k8sNamespace = getK8sNamespace();

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const inputPathParam = new GoString();
    inputPathParam["p"] = archive;
    inputPathParam["n"] = archive.length;

    const outputFileParam = new GoString();
    outputFileParam["p"] = outputFile;
    outputFileParam["n"] = outputFile.length;

    const downstreamsStr = JSON.stringify(downstreams)
    const downstreamsParam = new GoString();
    downstreamsParam["p"] = downstreamsStr;
    downstreamsParam["n"] = downstreamsStr.length;

    const k8sNamespaceParam = new GoString();
    k8sNamespaceParam["p"] = k8sNamespace;
    k8sNamespaceParam["n"] = k8sNamespace.length;

    const registryJson = JSON.stringify(registryInfo)
    const registryJsonParam = new GoString();
    registryJsonParam["p"] = registryJson;
    registryJsonParam["n"] = registryJson.length;

    const updatedConfigValuesParam = new GoString();
    updatedConfigValuesParam["p"] = updatedConfigValues;
    updatedConfigValuesParam["n"] = updatedConfigValues.length;

    kots().RewriteVersion(socketParam, inputPathParam, outputFileParam, downstreamsParam, k8sNamespaceParam, registryJsonParam, copyImages, app.isAirgap, updatedConfigValuesParam);

    let errrorMessage = "";

    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "running") {
        stores.kotsAppStore.setImageRewriteStatus(obj.display_message, "running");
        return false;
      }
      if (obj.status === "terminated") {
        if (obj.exit_code !== 0) {
          errrorMessage = obj.display_message;
        }
        resolve();
        return true;
      }
      return false;
    });

    if (errrorMessage) {
      await stores.kotsAppStore.setImageRewriteStatus(errrorMessage, "failed");
      throw new ReplicatedError(errrorMessage);
    }

    return "";

  } finally {
    tmpDir.removeCallback();
  }
}

export function getK8sNamespace(): string {
  if (process.env["DEV_NAMESPACE"]) {
    return String(process.env["DEV_NAMESPACE"]);
  }
  if (process.env["POD_NAMESPACE"]) {
    return String(process.env["POD_NAMESPACE"]);
  }
  return "default";
}

export function getKotsadmNamespace(): string {
  if (process.env["POD_NAMESPACE"]) {
    return String(process.env["POD_NAMESPACE"]);
  }
  return "default";
}
