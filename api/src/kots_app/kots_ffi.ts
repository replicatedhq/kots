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
  extractPreflightSpecFromTarball,
  extractSupportBundleSpecFromTarball,
  extractAppSpecFromTarball,
  extractKotsAppSpecFromTarball,
  extractAppTitleFromTarball,
  extractAppIconFromTarball,
  extractKotsAppLicenseFromTarball,
  extractAnalyzerSpecFromTarball,
} from "../util/tar";
import { KotsAppRegistryDetails } from "../kots_app"
import { Cluster } from "../cluster";
import * as _ from "lodash";
import yaml, { dump } from "js-yaml";
import { StatusServer } from "../airgap/status";
import { getDiffSummary } from "../util/utilities";
import { ReplicatedError } from "../server/errors";

const GoString = Struct({
  p: "string",
  n: "longlong"
});

function kots() {
  return ffi.Library("/lib/kots.so", {
    TestRegistryCredentials: ["void", [GoString, GoString, GoString, GoString, GoString]],
    PullFromLicense: ["void", [GoString, GoString, GoString, GoString]],
    PullFromAirgap: ["void", [GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString]],
    UpdateCheck: ["void", [GoString, GoString]],
    ListUpdates: ["void", [GoString, GoString, GoString]],
    UpdateDownload: ["void", [GoString, GoString, GoString]],
    ReadMetadata: ["void", [GoString, GoString]],
    RemoveMetadata: ["void", [GoString, GoString]],
    RewriteImagesInVersion: ["void", [GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString, GoString]],
    TemplateConfig: [GoString, [GoString, GoString, GoString]],
    EncryptString: [GoString, [GoString, GoString]],
    DecryptString: [GoString, [GoString, GoString]],
    GetLatestLicense: [GoString, [GoString, GoString]],
    VerifyAirgapLicense: [GoString, [GoString]],
  });
}

export async function kotsAppGetBranding(): Promise<string> {
  const namespace = process.env["POD_NAMESPACE"];
  if (!namespace) {
    throw new Error("unable to determine current namespace");
  }

  const tmpDir = tmp.dirSync();
  try {
    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const namespaceParam = new GoString();
    namespaceParam["p"] = namespace;
    namespaceParam["n"] = namespace.length;

    let branding = "";
    kots().ReadMetadata(socketParam, namespaceParam);

    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        branding = obj.data;
        if (obj.exit_code === 0) {
          resolve();
        } else {
          reject(new Error(`process failed: ${obj.display_message}`));
        }
        return true;
      }
      return false;
    });

    return branding;

  } finally {
    tmpDir.removeCallback();
  }
}

interface Update {
  cursor: string;
}

export async function kotsAppCheckForUpdates(app: KotsApp, currentCursor: string): Promise<Update[]> {
  // We need to include the last archive because if there is an update, the ffi function will update it
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive(""+(app.currentSequence!)));

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    const currentCursorParam = new GoString();
    currentCursorParam["p"] = currentCursor ? currentCursor : "";
    currentCursorParam["n"] = currentCursor ? currentCursor.length : 0;

    console.log(`Check for updates current cursor = ${currentCursor}`);

    kots().ListUpdates(socketParam, archiveParam, currentCursorParam);

    await statusServer.connection();
    const update: Update[] = await statusServer.termination((resolve, reject, obj): boolean => {
      if (obj.status === "terminated") {
        if (obj.exit_code === 0) {
          resolve(JSON.parse(obj.data) as Update[]);
        } else {
          reject(new Error(`process failed: ${obj.display_message}`));
        }
        return true;
      }
      return false;
    });
    if (update) {
      console.log(`Check for updates got updates ${JSON.stringify(update)}`);
      return update;
    }
    return [];
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsAppDownloadUpdates(updatesAvailable: Update[], app: KotsApp, stores: Stores): Promise<void> {
  for (let i = 0; i < updatesAvailable.length; i++) {
    const update = updatesAvailable[i];
    try {
      await kotsAppDownloadUpdate(update.cursor, app, stores);
    } catch (err) {
      console.error(`Failed to download release ${update.cursor}: ${err}`);
    }
  }
}

export async function kotsAppDownloadUpdate(cursor: string, app: KotsApp, stores: Stores): Promise<boolean> {
  // We need to include the last archive because if there is an update, the ffi function will update it
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive(""+(app.currentSequence!)));

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    const cursorParam = new GoString();
    cursorParam["p"] = cursor;
    cursorParam["n"] = cursor.length;

    kots().UpdateDownload(socketParam, archiveParam, cursorParam);
    await statusServer.connection();
    const isUpdateAvailable: number = await statusServer.termination((resolve, reject, obj): boolean => {
      if (obj.status === "terminated") {
        if (obj.exit_code !== -1) {
          resolve(obj.exit_code);
        } else {
          reject(new Error(`process failed: ${obj.display_message}`));
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
      await saveUpdateVersion(archive, app, stores);
    }

    return isUpdateAvailable > 0;
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsAppCheckForUpdate(currentCursor: string, app: KotsApp, stores: Stores): Promise<boolean> {
  // We need to include the last archive because if there is an update, the ffi function will update it
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");

  try {
    fs.writeFileSync(archive, await app.getArchive(""+(app.currentSequence!)));

    let isUpdateAvailable = -1;

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    kots().UpdateCheck(socketParam, archiveParam);
    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        isUpdateAvailable = obj.exit_code;
        if (obj.exit_code !== -1) {
          resolve();
        } else {
          reject(new Error(`process failed: ${obj.display_message}`));
        }
        return true;
      }
      return false;
    });

    if (isUpdateAvailable < 0) {
      console.log("error checking for updates")
      return false;
    }

    if (isUpdateAvailable > 0) {
      await saveUpdateVersion(archive, app, stores);
    }

    return isUpdateAvailable > 0;
  } finally {
    tmpDir.removeCallback();
  }
}

async function saveUpdateVersion(archive: string, app: KotsApp, stores: Stores) {
  // if there was an update available, expect that the new archive is in the smae place as the one we pased in
  const params = await Params.getParams();
  const buffer = fs.readFileSync(archive);
  const newSequence = (await stores.kotsAppStore.getMaxSequence(app.id)) + 1;
  const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, `${newSequence}.tar.gz`);
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);

  console.log(`Save new version ${app.id}:${newSequence}, cursor=${installationSpec.cursor}`);

  await stores.kotsAppStore.createMidstreamVersion(
    app.id,
    newSequence,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    kotsAppLicense,
    appTitle,
    appIcon
  );

  const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(app.id);
  for (const clusterId of clusterIds) {
    const diffSummary = await getDiffSummary(app);
    await stores.kotsAppStore.createDownstreamVersion(app.id, newSequence, clusterId, installationSpec.versionLabel, "pending", "Upstream Update", diffSummary);
  }
}

export async function kotsAppFromLicenseData(licenseData: string, name: string, downstreamName: string, stores: Stores): Promise<KotsApp> {
  const parsedLicense = yaml.safeLoad(licenseData);
  if (parsedLicense.spec.isAirgapSupported) {
    try {
      const kotsApp = await stores.kotsAppStore.getPendingKotsAirgapApp();
      await stores.kotsAppStore.updateKotsAppLicense(kotsApp.id, licenseData);
      return kotsApp;
    } catch(e) {
      console.log("no pending airgap install found, creating a new app");
    }

    const kotsApp = await stores.kotsAppStore.createKotsApp(name, `replicated://${parsedLicense.spec.appSlug}`, licenseData, parsedLicense.spec.isAirgapSupported);
    return kotsApp;
  }

  const kotsApp = await stores.kotsAppStore.createKotsApp(name, `replicated://${parsedLicense.spec.appSlug}`, licenseData, !!parsedLicense.spec.isAirgapSupported);
  await kotsFinalizeApp(kotsApp, downstreamName, stores);

  return kotsApp;
}

export async function kotsFinalizeApp(kotsApp: KotsApp, downstreamName: string, stores: Stores) {
  const tmpDir = tmp.dirSync();

  try {
    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const licenseDataParam = new GoString();
    licenseDataParam["p"] = kotsApp.license;
    licenseDataParam["n"] = String(kotsApp.license).length;

    const downstreamParam = new GoString();
    downstreamParam["p"] = downstreamName;
    downstreamParam["n"] = downstreamName.length;

    const out = path.join(tmpDir.name, "archive.tar.gz");
    const outParam = new GoString();
    outParam["p"] = out;
    outParam["n"] = out.length;

    kots().PullFromLicense(socketParam, licenseDataParam, downstreamParam, outParam);
    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        if (obj.exit_code === 0) {
          resolve();
        } else {
          reject(new Error(`process failed: ${obj.display_message}`));
        }
        return true;
      }
      return false;
    });

    const params = await Params.getParams();
    const buffer = fs.readFileSync(out);

    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const installationSpec = await extractInstallationSpecFromTarball(buffer);

    const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
    const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
    const preflightSpec = await extractPreflightSpecFromTarball(buffer);
    const appSpec = await extractAppSpecFromTarball(buffer);
    const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
    const appTitle = await extractAppTitleFromTarball(buffer);
    const appIcon = await extractAppIconFromTarball(buffer);
    const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);
    kotsApp.hasPreflight = !!preflightSpec;

    await stores.kotsAppStore.createMidstreamVersion(
      kotsApp.id,
      0,
      installationSpec.versionLabel,
      installationSpec.releaseNotes,
      installationSpec.cursor,
      installationSpec.encryptionKey,
      supportBundleSpec,
      analyzersSpec,
      preflightSpec,
      appSpec,
      kotsAppSpec,
      kotsAppLicense,
      appTitle,
      appIcon
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

      const downstreamState = kotsApp.hasPreflight
        ? "pending_preflight"
        : "deployed";
      const diffSummary = await getDiffSummary(kotsApp);

      await stores.kotsAppStore.createDownstream(kotsApp.id, downstream, cluster.id);
      await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, installationSpec.versionLabel, downstreamState, "Kots Install", diffSummary);
    }

    kotsApp.currentSequence = 0;

    return kotsApp;
  } finally {
    tmpDir.removeCallback();
  }
}

export function kotsPullFromAirgap(socket: string, out: string, app: KotsApp, licenseData: string, airgapDir: string, downstreamName: string, stores: Stores, registryHost: string, registryNamespace: string, username: string, password: string): any {
  const socketParam = new GoString();
  socketParam["p"] = socket;
  socketParam["n"] = socket.length;

  const licenseDataParam = new GoString();
  licenseDataParam["p"] = licenseData;
  licenseDataParam["n"] = licenseData.length;

  const downstreamParam = new GoString();
  downstreamParam["p"] = downstreamName;
  downstreamParam["n"] = downstreamName.length;

  const airgapDirParam = new GoString();
  airgapDirParam["p"] = airgapDir;
  airgapDirParam["n"] = airgapDir.length;

  const outParam = new GoString();
  outParam["p"] = out;
  outParam["n"] = out.length;

  const registryHostParam = new GoString();
  registryHostParam["p"] = registryHost;
  registryHostParam["n"] = registryHost.length;

  const registryNamespaceParam = new GoString();
  registryNamespaceParam["p"] = registryNamespace;
  registryNamespaceParam["n"] = registryNamespace.length;

  const usernameParam = new GoString();
  usernameParam["p"] = username;
  usernameParam["n"] = username.length;

  const passwordParam = new GoString();
  passwordParam["p"] = password;
  passwordParam["n"] = password.length;

  kots().PullFromAirgap(socketParam, licenseDataParam, airgapDirParam, downstreamParam, outParam, registryHostParam, registryNamespaceParam, usernameParam, passwordParam);

  // args are returned so they are not garbage collected before native code is done
  return {
    socketParam,
    licenseDataParam,
    downstreamParam,
    airgapDirParam,
    outParam,
    registryHostParam,
    registryNamespaceParam,
    usernameParam,
    passwordParam,
  };
}

export async function kotsAppFromAirgapData(out: string, app: KotsApp, stores: Stores): Promise<{ hasPreflight: Boolean, isConfigurable: Boolean }> {
  const params = await Params.getParams();
  const buffer = fs.readFileSync(out);
  const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, "0.tar.gz");
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);

  await stores.kotsAppStore.createMidstreamVersion(
    app.id,
    0,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    kotsAppLicense,
    appTitle,
    appIcon
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

    const diffSummary = await getDiffSummary(app);

    await stores.kotsAppStore.createDownstream(app.id, downstream, cluster.id);
    await stores.kotsAppStore.createDownstreamVersion(app.id, 0, cluster.id, installationSpec.versionLabel, "deployed", "Airgap", diffSummary);
  }

  await stores.kotsAppStore.setKotsAirgapAppInstalled(app.id);

  app.currentSequence = 0;
  const isConfigurable = await app.isAppConfigurable();

  return {
    hasPreflight: !!preflightSpec,
    isConfigurable,
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

export async function kotsTemplateConfig(configPath: string, configContent: string, configValuesContent: string): Promise<any> {
  const configPathParam = new GoString();
  configPathParam["p"] = configPath;
  configPathParam["n"] = String(configPath).length;

  const configDataParam = new GoString();
  configDataParam["p"] = configContent;
  configDataParam["n"] = String(configContent).length;

  const configValuesDataParam = new GoString();
  configValuesDataParam["p"] = configValuesContent;
  configValuesDataParam["n"] = String(configValuesContent).length;

  const templatedConfig = kots().TemplateConfig(configPathParam, configDataParam, configValuesDataParam);
  if (templatedConfig == "" || templatedConfig["p"] == "") {
    throw new ReplicatedError("failed to template config");
  }

  try {
    return yaml.safeLoad(templatedConfig["p"]);
  } catch(err) {
    throw new ReplicatedError(`Failed to parse templated config ${err}`);
  }
}

export async function kotsEncryptString(cipherString: string, message: string): Promise<string> {
  const cipherStringParam = new GoString();
  cipherStringParam["p"] = cipherString;
  cipherStringParam["n"] = String(cipherString).length;

  const messageParam = new GoString();
  messageParam["p"] = message;
  messageParam["n"] = String(message).length;

  const encrypted = kots().EncryptString(cipherStringParam, messageParam);

  if (encrypted["p"] === null) {
    throw new ReplicatedError("Failed to encrypt string via FFI call");
  }

  return encrypted["p"];
}

export async function kotsDecryptString(cipherString: string, message: string): Promise<string> {
  const cipherStringParam = new GoString();
  cipherStringParam["p"] = cipherString;
  cipherStringParam["n"] = String(cipherString).length;

  const messageParam = new GoString();
  messageParam["p"] = message;
  messageParam["n"] = String(message).length;

  const decrypted = kots().DecryptString(cipherStringParam, messageParam);

  if (decrypted["p"] === null) {
    throw new ReplicatedError("Failed to encrypt string via FFI call");
  }

  return decrypted["p"];
}

export async function getLatestLicense(licenseData: string): Promise<string> {
  const tmpDir = tmp.dirSync();
  try {
    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const socketParam = new GoString();
    socketParam["p"] = statusServer.socketFilename;
    socketParam["n"] = statusServer.socketFilename.length;

    const licenseDataParam = new GoString();
    licenseDataParam["p"] = licenseData;
    licenseDataParam["n"] = String(licenseData).length;

    kots().GetLatestLicense(socketParam, licenseDataParam);

    let license = "";
    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "terminated") {
        license = obj.data;
        if (obj.exit_code !== -1) {
          resolve();
        } else {
          reject(new ReplicatedError("failed to get latest license"));
        }
        return true;
      }
      return false;
    });

    return license;
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsRewriteImagesInVersion(app: KotsApp, downstreams: string[], registryInfo: KotsAppRegistryDetails, outputFile: string, stores: Stores): Promise<string> {
  const tmpDir = tmp.dirSync();
  try {
    const k8sNamespace = getK8sNamespace();

    const statusServer = new StatusServer();
    await statusServer.start(tmpDir.name);

    const archive = path.join(tmpDir.name, "archive.tar.gz");
    fs.writeFileSync(archive, await app.getArchive(""+(app.currentSequence!)));
  
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

    const registryParam = new GoString();
    registryParam["p"] = registryInfo.registryHostname;
    registryParam["n"] = registryInfo.registryHostname.length;

    const usernamedParam = new GoString();
    usernamedParam["p"] = registryInfo.registryUsername;
    usernamedParam["n"] = registryInfo.registryUsername.length;

    const passwordParam = new GoString();
    passwordParam["p"] = registryInfo.registryPassword;
    passwordParam["n"] = registryInfo.registryPassword.length;

    const namespaceParam = new GoString();
    namespaceParam["p"] = registryInfo.namespace;
    namespaceParam["n"] = registryInfo.namespace.length;

    kots().RewriteImagesInVersion(socketParam, inputPathParam, outputFileParam, downstreamsParam, k8sNamespaceParam, registryParam, usernamedParam, passwordParam, namespaceParam);

    let errrorMessage = "";

    await statusServer.connection();
    await statusServer.termination((resolve, reject, obj): boolean => {
      // Return true if completed
      if (obj.status === "running") {
        Promise.all([stores.kotsAppStore.setImageRewriteStatus(obj.display_message, "running")]);
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

    await stores.kotsAppStore.setImageRewriteStatus("Generating new version", "running");

    const params = await Params.getParams();
    const buffer = fs.readFileSync(outputFile);
    const newSequence = app.currentSequence! + 1;
    const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, `${newSequence}.tar.gz`);
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const installationSpec = await extractInstallationSpecFromTarball(buffer);
    const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
    const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
    const preflightSpec = await extractPreflightSpecFromTarball(buffer);
    const appSpec = await extractAppSpecFromTarball(buffer);
    const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
    const appTitle = await extractAppTitleFromTarball(buffer);
    const appIcon = await extractAppIconFromTarball(buffer);
    const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);

    await stores.kotsAppStore.createMidstreamVersion(
      app.id,
      newSequence,
      installationSpec.versionLabel,
      installationSpec.releaseNotes,
      installationSpec.cursor,
      installationSpec.encryptionKey,
      supportBundleSpec,
      analyzersSpec,
      preflightSpec,
      appSpec,
      kotsAppSpec,
      kotsAppLicense,
      appTitle,
      appIcon
    );

    const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(app.id);
    for (const clusterId of clusterIds) {
      const diffSummary = await getDiffSummary(app);
      await stores.kotsAppStore.createDownstreamVersion(app.id, newSequence, clusterId, installationSpec.versionLabel, "pending", "Upstream Update", diffSummary);
    }

    await stores.kotsAppStore.clearImageRewriteStatus();

    return "";

  } finally {
    // tmpDir.removeCallback();
  }
}

export async function verifyAirgapLicense(licenseData: string): Promise<boolean> {
  const licenseDataParam = new GoString();
  licenseDataParam["p"] = licenseData;
  licenseDataParam["n"] = String(licenseData).length;

  const license = kots().VerifyAirgapLicense(licenseDataParam);
  if (license == "" || license["p"] == null) {
    throw new ReplicatedError("failed to verify airgap license signature");
  }

  return license["p"] === "verified";
}

export function getK8sNamespace(): String {
  if (process.env["DEV_NAMESPACE"]) {
    return String(process.env["DEV_NAMESPACE"]);
  }
  if (process.env["POD_NAMESPACE"]) {
    return String(process.env["POD_NAMESPACE"]);
  }
  return "default";
}
