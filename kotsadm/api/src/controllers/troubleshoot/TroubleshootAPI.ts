import { Request, Response } from "express";
import { Controller, Post, Put, Get, Res, Req, BodyParams, PathParams, QueryParams } from "@tsed/common";
import { Params } from "../../server/params";
import { logger } from "../../server/logger";
import { analyzeSupportBundle } from "../../troubleshoot/troubleshoot_ffi";
import fs from "fs";
import path from "path";
import { putObject, getS3 } from "../../util/s3";
import { Session } from "../../session";
import { Stores } from "../../schema/stores";

interface ErrorResponse {
  error: {};
}

interface BundleUploadedBody {
  size: number;
}

@Controller("/api/v1/troubleshoot")
export class TroubleshootAPI {

  @Post("/analyzebundle/:supportBundleId")
  public async analyzeBundle(
    @Res() response: Response,
    @Req() request: Request,
    @PathParams("supportBundleId") supportBundleId: string,
  ): Promise<any> {
    const stores = request.app.locals.stores;

    // Check if bundle exists
    const exists = await stores.troubleshootStore.supportBundleExists(supportBundleId);
    if (!exists) {
      response.send(404, "Bundle does not exist");
      return;
    }

    const b = await stores.troubleshootStore.getSupportBundle(supportBundleId);
    const analyzers = await stores.troubleshootStore.tryGetAnalyzersForKotsApp(b.watchId);
    await performAnalysis(supportBundleId, analyzers, stores);
    const analyzedBundle = await stores.troubleshootStore.getSupportBundle(supportBundleId);

    response.send(200, analyzedBundle);
  }

  @Post("/:watchId/:supportBundleId")
  public async bundleUploaded(
    @Res() response: Response,
    @Req() request: Request,
    @BodyParams("") body: BundleUploadedBody,
    @PathParams("watchId") watchId: string,
    @PathParams("supportBundleId") supportBundleId: string,
  ): Promise<any> {
    const stores = request.app.locals.stores;

    // Don't create support bundle if there is one with the same ID
    const exists = await stores.troubleshootStore.supportBundleExists(supportBundleId);
    if (exists) {
      response.send(403);
      return;
    }

    const fileInfo = await stores.troubleshootStore.getSupportBundleFileInfo(supportBundleId);

    logger.debug({ msg: `creating support bundle record with id ${supportBundleId} via upload callback` });

    await stores.troubleshootStore.createSupportBundle(watchId, fileInfo.ContentLength, supportBundleId);
    await performAnalysis(supportBundleId, "", stores);

    response.send(204, "");
  }

}

async function performAnalysis(supportBundleId: string, analyzers: string, stores: Stores) {
  await stores.troubleshootStore.markSupportBundleUploaded(supportBundleId);
  const supportBundle = await stores.troubleshootStore.getSupportBundle(supportBundleId);
  const dirTree = await supportBundle.generateFileTreeIndex();
  await stores.troubleshootStore.assignTreeIndex(supportBundleId, JSON.stringify(dirTree));

  await analyzeSupportBundle(supportBundleId, analyzers, stores);
}

async function s3getBundle(bundleId, response) {
  const replicatedParams = await Params.getParams();
  const params = {
    Bucket: replicatedParams.shipOutputBucket,
    Key: `${replicatedParams.s3BucketEndpoint !== "" ? `${replicatedParams.shipOutputBucket}/` : ""}supportbundles/${bundleId}/supportbundle.tar.gz`,
  };

  return new Promise((resolve, reject) => {
    response.on("error", err => {
      console.log(err);
      resolve(false);
    });

    response.on("finish", async () => {
      try {
        resolve(true);
      } catch (err) {
        console.log(err);
        resolve(false);
      }
    });

    getS3(replicatedParams).getObject(params).createReadStream().pipe(response);
  });
}
