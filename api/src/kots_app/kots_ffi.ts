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
  extractCursorAndVersionFromTarball,
  extractPreflightSpecFromTarball
} from "../util/tar";
import { Cluster } from "../cluster";
import * as _ from "lodash";
import yaml from "js-yaml";

const GoString = Struct({
  p: "string",
  n: "longlong"
});

function kots() {
  return ffi.Library("/lib/kots.so", {
    PullFromLicense: ["longlong", [GoString, GoString, GoString]],
    PullFromAirgap: ["longlong", [GoString, GoString, GoString, GoString, GoString, GoString]],
    RewriteAndPushImageName: ["void", [GoString, GoString, GoString, GoString, GoString, GoString, GoString]],
    UpdateCheck: ["longlong", [GoString]],
    ReadMetadata: [GoString, [GoString]],
    RemoveMetadata: ["longlong", [GoString]],
  });
}

export async function kotsAppGetBranding(): Promise<string> {
  const namespace = process.env["POD_NAMESPACE"];
  if (!namespace) {
    throw new Error("unable to determine current namespace");
  }
  const namespaceParam = new GoString();
  namespaceParam["p"] = namespace;
  namespaceParam["n"] = namespace.length;

  const branding = kots().ReadMetadata(namespaceParam);
  return branding.p;
}

export async function kotsAppCheckForUpdate(currentCursor: string, app: KotsApp, stores: Stores): Promise<boolean> {
  // We need to include the last archive because if there is an update, the ffi function will update it
  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "archive.tar.gz");
  try {
    fs.writeFileSync(archive, await app.getArchive(""+(app.currentSequence!)));

    const archiveParam = new GoString();
    archiveParam["p"] = archive;
    archiveParam["n"] = archive.length;

    const isUpdateAvailable = kots().UpdateCheck(archiveParam);

    if (isUpdateAvailable < 0) {
      console.log("error checking for updates")
      return false;
    }

    if (isUpdateAvailable > 0) {
      // if there was an update available, expect that the new archive is in the smae place as the one we pased in
      const params = await Params.getParams();
      const buffer = fs.readFileSync(archive);
      const newSequence = app.currentSequence! + 1;
      const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, `${newSequence}.tar.gz`);
      await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

      const cursorAndVersion = await extractCursorAndVersionFromTarball(buffer);
      const preflightSpec = await extractPreflightSpecFromTarball(buffer);
      await stores.kotsAppStore.createMidstreamVersion(app.id, newSequence, cursorAndVersion.versionLabel, cursorAndVersion.cursor, undefined, preflightSpec);

      const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(app.id);
      for (const clusterId of clusterIds) {
        await stores.kotsAppStore.createDownstreamVersion(app.id, newSequence, clusterId, cursorAndVersion.versionLabel);
      }
    }

    return isUpdateAvailable > 0;
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsAppFromLicenseData(licenseData: string, name: string, downstreamName: string, stores: Stores): Promise<KotsApp | void> {
  const parsedLicense = yaml.safeLoad(licenseData);
  if (parsedLicense.spec.isAirgapSupported) {
    try {
      const kotsApp = await stores.kotsAppStore.getPendingKotsAirgapApp();
      return kotsApp;
    } catch(e) {
      console.log("no pending airgap install found, creating a new app");
    }

    const kotsApp = await stores.kotsAppStore.createKotsApp(name, `replicated://${parsedLicense.spec.appSlug}`, licenseData, parsedLicense.spec.isAirgapSupported);
    return kotsApp;
  }

  const kotsApp = await stores.kotsAppStore.createKotsApp(name, `replicated://${parsedLicense.spec.appSlug}`, licenseData, !!parsedLicense.spec.isAirgapSupported);
  await kotsFinalizeApp(kotsApp, downstreamName, stores)

  return kotsApp;
}

export async function kotsFinalizeApp(kotsApp: KotsApp, downstreamName: string, stores: Stores) {
  const tmpDir = tmp.dirSync();

  try {
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

    const pullResult = kots().PullFromLicense(licenseDataParam, downstreamParam, outParam);
    if (pullResult > 0) {
      return;
    }

    const params = await Params.getParams();
    const buffer = fs.readFileSync(out);

    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const cursorAndVersion = await extractCursorAndVersionFromTarball(buffer);

    const preflightSpec = await extractPreflightSpecFromTarball(buffer);
    kotsApp.hasPreflight = !!preflightSpec;

    await stores.kotsAppStore.createMidstreamVersion(kotsApp.id, 0, cursorAndVersion.versionLabel, cursorAndVersion.cursor, undefined, preflightSpec);

    const downstreams = await extractDownstreamNamesFromTarball(buffer);
    const clusters = await stores.clusterStore.listAllUsersClusters();
    for (const downstream of downstreams) {
      const cluster = _.find(clusters, (c: Cluster) => {
        return c.title === downstream;
      });

      if (!cluster) {
        continue;
      }

      await stores.kotsAppStore.createDownstream(kotsApp.id, downstream, cluster.id);
      await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, cursorAndVersion.versionLabel);
    }

    return kotsApp;
  } catch (err) {
    console.log(err);
  } finally {
    tmpDir.removeCallback();
  }
}

export async function kotsAppFromAirgapData(app: KotsApp, licenseData: string, airgapDir: string, downstreamName: string, stores: Stores, registryHost: string, registryNamespace: string): Promise<{ hasPreflight: Boolean} | undefined> {
  const tmpDstDir = tmp.dirSync();

  try {
    const licenseDataParam = new GoString();
    licenseDataParam["p"] = licenseData;
    licenseDataParam["n"] = licenseData.length;

    const downstreamParam = new GoString();
    downstreamParam["p"] = downstreamName;
    downstreamParam["n"] = downstreamName.length;

    const airgapDirParam = new GoString();
    airgapDirParam["p"] = airgapDir;
    airgapDirParam["n"] = airgapDir.length;

    const out = path.join(tmpDstDir.name, "archive.tar.gz");
    const outParam = new GoString();
    outParam["p"] = out;
    outParam["n"] = out.length;

    const registryHostParam = new GoString();
    registryHostParam["p"] = registryHost;
    registryHostParam["n"] = registryHost.length;

    const registryNamespaceParam = new GoString();
    registryNamespaceParam["p"] = registryNamespace;
    registryNamespaceParam["n"] = registryNamespace.length;

    const pullResult = kots().PullFromAirgap(licenseDataParam, airgapDirParam, downstreamParam, outParam, registryHostParam, registryNamespaceParam);
    if (pullResult > 0) {
      return;
    }

    const params = await Params.getParams();
    const buffer = fs.readFileSync(out);
    const objectStorePath = path.join(params.shipOutputBucket.trim(), app.id, "0.tar.gz");
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const cursorAndVersion = await extractCursorAndVersionFromTarball(buffer);
    const preflightSpec = await extractPreflightSpecFromTarball(buffer);
    await stores.kotsAppStore.createMidstreamVersion(app.id, 0, cursorAndVersion.versionLabel, cursorAndVersion.cursor, undefined, preflightSpec);

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
      await stores.kotsAppStore.createDownstreamVersion(app.id, 0, cluster.id, cursorAndVersion.versionLabel);
    }

    await stores.kotsAppStore.setKotsAirgapAppInstalled(app.id);
    // NOTE: Finally statement will still run, despite try {} block returning
    return {
      hasPreflight: !!preflightSpec
    };
  } finally {
    tmpDstDir.removeCallback();
  }
}

export function kotsRewriteAndPushImageName(socket: string, imageFile: string, image: string, registryHost: string, registryOrg: string, username: string, password: string): void {
  const socketParam = new GoString();
  socketParam["p"] = socket;
  socketParam["n"] = socket.length;

  const imageFileParam = new GoString();
  imageFileParam["p"] = imageFile;
  imageFileParam["n"] = imageFile.length;

  const imageParam = new GoString();
  imageParam["p"] = image;
  imageParam["n"] = image.length;

  const registryHostParam = new GoString();
  registryHostParam["p"] = registryHost;
  registryHostParam["n"] = registryHost.length;

  const registryOrgParam = new GoString();
  registryOrgParam["p"] = registryOrg;
  registryOrgParam["n"] = registryOrg.length;

  const usernameParam = new GoString();
  usernameParam["p"] = username;
  usernameParam["n"] = username.length;

  const passwordParam = new GoString();
  passwordParam["p"] = password;
  passwordParam["n"] = password.length;

  kots().RewriteAndPushImageName(socketParam, imageFileParam, imageParam, registryHostParam, registryOrgParam, usernameParam, passwordParam);
}
