import ffi from "ffi";
import Struct from "ref-struct";
import { Stores } from "../schema/stores";
import * as _ from "lodash";
import path from "path";
import tmp from "tmp";
import { Params } from "../server/params";
import { getS3 } from "../util/s3";
import fs from "fs";

const GoString = Struct({
  p: "string",
  n: "longlong"
});

function troubleshoot() {
  return ffi.Library("/lib/troubleshoot.so", {
    Analyze: [GoString, [GoString, GoString, GoString]],
  });
}

export async function analyzeSupportBundle(supportBundleId: string, stores: Stores): Promise<boolean> {
  // const supportBundle = await stores.troubleshootStore.getSupportBundle(supportBundleId);

  // Download the support bundle to a temp file
  // and pass that into the analyze function

  const tmpDir = tmp.dirSync();
  const archive = path.join(tmpDir.name, "support-bundle.tar.gz");

  const replicatedParams = await Params.getParams();
  const s3Params = {
    Bucket: replicatedParams.shipOutputBucket,
    Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}supportbundles/${supportBundleId}/supportbundle.tar.gz`,
  };

  return new Promise((resolve, reject) => {
    const writeStream = fs.createWriteStream(archive);

    writeStream.on("error", err => {
      console.log(err);
      resolve(false);
    });

    writeStream.on("finish", async () => {
      try {
        const bundleURLParam = new GoString();
        bundleURLParam["p"] = archive;
        bundleURLParam["n"] = archive.length;

        const outputFormatParam = new GoString();
        outputFormatParam["p"] = "json";
        outputFormatParam["n"] = outputFormatParam["p"].length;

        const compatibilityParam = new GoString();
        compatibilityParam["p"] = "support-bundle";
        compatibilityParam["n"] = compatibilityParam["p"].length;

        const analysisResult = troubleshoot().Analyze(bundleURLParam, outputFormatParam, compatibilityParam);
        if (analysisResult == "" || analysisResult["p"] == "") {
          await stores.troubleshootStore.updateSupportBundleStatus(supportBundleId, "analysis_error");
          console.log("failed to analyze");
          return false;
        }

        await stores.troubleshootStore.setAnalysisResult(supportBundleId, analysisResult["p"]);
        await stores.troubleshootStore.updateSupportBundleStatus(supportBundleId, "analyzed");

        resolve(true);
      } catch (err) {
        console.log(err);
        resolve(false);
      }
    });

    getS3(replicatedParams).getObject(s3Params).createReadStream().pipe(writeStream);
  });
 }
