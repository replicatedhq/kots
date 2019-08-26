import ffi from "ffi";
import Struct from "ref-struct";
import { Stores } from "../schema/stores";
import { KotsApp } from "./";
import { Params } from "../server/params";
import { putObject } from "../util/s3";
import path from "path";
import tmp from "tmp";
import fs from "fs";
import { extractDownstreamNamesFromTarball } from "../util/tar";
import { Cluster } from "../cluster";
import * as _ from "lodash";

const GoString = Struct({
  p: "string",
  n: "longlong"
});

function kots() {
  return ffi.Library("./kots.so", {
    PullFromLicense: [GoString, [GoString, GoString, GoString]],
  });
}

export async function kotsAppFromLicenseData(licenseData: string, name: string, downstreamName: string, stores: Stores): Promise<KotsApp | void> {
  const tmpDir = tmp.dirSync();

  try {
    const licenseDataParam = new GoString();
    licenseDataParam["p"] = licenseData;
    licenseDataParam["n"] = licenseData.length;

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

    const kotsApp = await stores.kotsAppStore.createKotsApp(name, "replicated://sentry-enterprise", licenseData);
    await stores.kotsAppStore.createKotsAppVersion(kotsApp.id, 0, "??", "0", undefined, undefined);

    const params = await Params.getParams();
    const buffer = fs.readFileSync(out);
    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

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
    }

    return kotsApp;
  } catch (err) {
    console.log(err);
  } finally {
    tmpDir.removeCallback();
  }
}
