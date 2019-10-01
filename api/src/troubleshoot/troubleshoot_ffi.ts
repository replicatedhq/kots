import ffi from "ffi";
import Struct from "ref-struct";
import { Stores } from "../schema/stores";
import * as _ from "lodash";

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
  const supportBundle = await stores.troubleshootStore.getSupportBundle(supportBundleId);
  const presignedDownloadURL = await stores.troubleshootStore.signSupportBundleGetRequest(supportBundle);

  try {
    const bundleURLParam = new GoString();
    bundleURLParam["p"] = presignedDownloadURL;
    bundleURLParam["n"] = presignedDownloadURL.length;

    const outputFormatParam = new GoString();
    outputFormatParam["p"] = "json";
    outputFormatParam["n"] = outputFormatParam["p"].length;

    const compatibilityParam = new GoString();
    compatibilityParam["p"] = "support-bundle";
    compatibilityParam["n"] = compatibilityParam["p"].length;

    const analysisResult = troubleshoot().Analyze(bundleURLParam, outputFormatParam, compatibilityParam);
    if (analysisResult == "" || analysisResult["p"] == "") {
      await stores.troubleshootStore.updateSupportBundleStatus(supportBundleId, "analysis_error");
      return false;
    }

    await stores.troubleshootStore.setAnalysisResult(supportBundleId, analysisResult["p"]);
    await stores.troubleshootStore.updateSupportBundleStatus(supportBundleId, "analyzed");

    return true;
  } catch (err) {
    console.log(err);
    return false;
  }
}
