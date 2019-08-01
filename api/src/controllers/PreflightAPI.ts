import { Request, Response } from "express";
import { Controller, Get, PathParams, Put, Res, Req} from "ts-express-decorators";
import jsYaml from "js-yaml";
import { Params } from "../server/params";

@Controller("/api/v1/preflight")
export class PreflightAPI {
  @Get("/:watchId/:downstream_cluster") // can we use watchSlug instead?
  async getPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    @PathParams("downstream_cluster") downstreamCluster: string
  ): Promise<void> {
    // Fetch YAML from the database and return to client with injected key
    const preflightSpec: string = await request.app.locals.stores.preflightStore.getPreflightSpec(watchId);
    if (!preflightSpec) {
      response.send(404);
    }

    const params = await Params.getParams();
    const putUrl = `https://${params.apiAdvertiseEndpoint}/api/v1/preflight/${watchId}/${downstreamCluster}`
    const parsedPreflightSpec = jsYaml.load(preflightSpec);
    parsedPreflightSpec.spec.sendResultsTo(putUrl);

    response.setHeader("Content-Type", "text/x-yaml");
    response.send(200, preflightSpec.toString());
  }

  @Put("/:watchId/:downstream_cluster")
  async putPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    // @PathParams("downstream_cluster") downstream_cluster: string
  ): Promise<void> {

    // Write preflight results to the database
    const result = request.body;
    await request.app.locals.stores.preflightStore.addPreflightResult(watchId, result);
    response.send(200);
  }
}
