import Express from "express";
import { Controller, Post, Get, Res, Req, BodyParams, PathParams } from "@tsed/common";
import { Params } from "../server/params";
import { logger } from "../server/logger";
import jsYaml from "js-yaml";

interface ErrorResponse {
  error: {};
}

interface BundleUploadedBody {
  size: number;
}

@Controller("/api/v1/troubleshoot")
export class TroubleshootAPI {
  @Get("/:partA/:partB")
  async getSpec(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("partA") partA: string,
    @PathParams("partB") partB: string,
  ): Promise<any | ErrorResponse> {

    const slug = partA + "/" + partB;
    const collector = await request.app.locals.stores.troubleshootStore.getCollectorForWatchSlug(slug);
    const watchId = await request.app.locals.stores.watchStore.getIdFromSlug(slug);
    const supportBundle = await request.app.locals.stores.troubleshootStore.getBlankSupportBundle(watchId);

    const uploadUrl = await request.app.locals.stores.troubleshootStore.signSupportBundlePutRequest(supportBundle);

    const params = await Params.getParams();
    const callbackUrl = `${params.apiAdvertiseEndpoint}/api/v1/troubleshoot/${watchId}/${supportBundle.id}`;

    const parsedSpec = jsYaml.load(collector.spec);
    parsedSpec.spec.afterCollection =  [
      { "uploadResultsTo": {"method": "PUT", "uri": uploadUrl} },
      { "callback": {"method": "POST", "uri": callbackUrl} },
    ];
    response.send(200, parsedSpec);
  }

  @Post("/:watchId/:supportBundleId")
  public async bundleUploaded(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @BodyParams("") body: BundleUploadedBody,
    @PathParams("watchId") watchId: string,
    @PathParams("supportBundleId") supportBundleId: string,
  ): Promise<any | ErrorResponse> {

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

    response.send(204, "");
  }
}
