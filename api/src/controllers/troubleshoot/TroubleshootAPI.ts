import { Request, Response } from "express";
import { Controller, Post, Put, Get, Res, Req, BodyParams, PathParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";
import { Params } from "../../server/params";
import { logger } from "../../server/logger";
import jsYaml from "js-yaml";
import { TroubleshootStore } from "../../troubleshoot";
import { analyzeSupportBundle } from "../../troubleshoot/troubleshoot_ffi";
import fs from "fs";
import path from "path";
import { putObject } from "../../util/s3";

interface ErrorResponse {
  error: {};
}

interface BundleUploadedBody {
  size: number;
}

@Controller("/api/v1/troubleshoot")
export class TroubleshootAPI {
  @Get("/:slug")
  async getSpec(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("slug") slug: string,
  ): Promise<any | ErrorResponse> {
    let collector = TroubleshootStore.defaultSpec;

    const kotsCollector = await request.app.locals.stores.troubleshootStore.tryGetCollectorForKotsSlug(slug);
    if (kotsCollector) {
      collector = kotsCollector;
    } else {
      const watchCollector = await request.app.locals.stores.troubleshootStore.tryGetCollectorForWatchSlug(slug);
      if (watchCollector) {
        collector = watchCollector;
      }
    }

    let appOrWatchId;

    try {
      appOrWatchId = await request.app.locals.stores.kotsAppStore.getIdFromSlug(slug);
    } catch {
      appOrWatchId = await request.app.locals.stores.watchStore.getIdFromSlug(slug);
    }

    const supportBundle = await request.app.locals.stores.troubleshootStore.getBlankSupportBundle(appOrWatchId);

    const params = await Params.getParams();
    const uploadUrl = `${params.apiAdvertiseEndpoint}/api/v1/troubleshoot/${appOrWatchId}/${supportBundle.id}`;

    const parsedSpec = jsYaml.load(collector);
    parsedSpec.spec.afterCollection = [
      {
        "uploadResultsTo": {
          "method": "PUT",
          "uri": uploadUrl,
        },
      },
    ];

    response.send(200, parsedSpec);
  }

  @Put("/:watchId/:supportBundleId")
  public async bundleUpload(
    @Res() response: Response,
    @Req() request: Request,
    @PathParams("watchId") watchId: string,
    @PathParams("supportBundleId") supportBundleId: string,
  ): Promise<any> {
    const bundleFile = request.app.locals.bundleFile;

    try {
      const exists = await request.app.locals.stores.troubleshootStore.supportBundleExists(supportBundleId);
      if (exists) {
        response.send(403);
        return
      }

      // upload it to s3
      const params = await Params.getParams();
      const buffer = fs.readFileSync(bundleFile);
      await putObject(params, path.join(params.shipOutputBucket.trim(), "supportbundles", supportBundleId, "supportbundle.tar.gz"), buffer, params.shipOutputBucket);
      const fileInfo = await request.app.locals.stores.troubleshootStore.getSupportBundleFileInfo(supportBundleId);

      logger.debug({msg: `creating support bundle record with id ${supportBundleId} via upload callback`});

      await request.app.locals.stores.troubleshootStore.createSupportBundle(watchId, fileInfo.ContentLength, supportBundleId);
      await request.app.locals.stores.troubleshootStore.markSupportBundleUploaded(supportBundleId);

      const supportBundle = await request.app.locals.stores.troubleshootStore.getSupportBundle(supportBundleId);
      const dirTree = await supportBundle.generateFileTreeIndex();
      await request.app.locals.stores.troubleshootStore.assignTreeIndex(supportBundleId, JSON.stringify(dirTree));

      // Analyze it
      await analyzeSupportBundle(supportBundleId, request.app.locals.stores);
    } finally {
      fs.unlinkSync(bundleFile);
    }
  }

  @Post("/:watchId/:supportBundleId")
  public async bundleUploaded(
    @Res() response: Response,
    @Req() request: Request,
    @BodyParams("") body: BundleUploadedBody,
    @PathParams("watchId") watchId: string,
    @PathParams("supportBundleId") supportBundleId: string,
  ): Promise<any> {

    // Don't ctreate support bundle if there is one with the same ID
    const exists = await request.app.locals.stores.troubleshootStore.supportBundleExists(supportBundleId);
    if (exists) {
      response.send(403);
      return
    }

    const fileInfo = await request.app.locals.stores.troubleshootStore.getSupportBundleFileInfo(supportBundleId);

    logger.debug({msg: `creating support bundle record with id ${supportBundleId} via upload callback`});

    await request.app.locals.stores.troubleshootStore.createSupportBundle(watchId, fileInfo.ContentLength, supportBundleId);
    await request.app.locals.stores.troubleshootStore.markSupportBundleUploaded(supportBundleId);

    const supportBundle = await request.app.locals.stores.troubleshootStore.getSupportBundle(supportBundleId);
    const dirTree = await supportBundle.generateFileTreeIndex();
    await request.app.locals.stores.troubleshootStore.assignTreeIndex(supportBundleId, JSON.stringify(dirTree));

    // Analyze it
    await analyzeSupportBundle(supportBundleId, request.app.locals.stores);

    response.send(204, "");
  }
}
