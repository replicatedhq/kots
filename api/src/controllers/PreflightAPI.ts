import { Request, Response } from "express";
import { Controller, Get, PathParams, Put, Res, Req} from "ts-express-decorators";
import jsYaml from "js-yaml";

import { Params } from "../server/params";

@Controller("/api/v1/preflight")
export class PreflightAPI {
  @Get("/*")
  async getPreflightStatus(
    @Req() request: Request,
    @Res() response: Response

  ): Promise<void> {
    const splitPath = request.path.split('/').slice(1);
    let watchSlug;
    let clusterSlug;
    console.log(splitPath);
    if (splitPath.length === 3) {
      watchSlug = splitPath.slice(0, 2).join('/');
      clusterSlug = splitPath[2];
    } else if (splitPath.length === 2) {
      watchSlug = splitPath[0];
      clusterSlug = splitPath[1];
    } else {
      response.send(400);
      return;
    }

    console.log('watchSlug', watchSlug);
    console.log('clusterSlug', clusterSlug);

    try {
      // Fetch YAML from the database and return to client with injected key
      const preflightSpec = await request.app.locals.stores.preflightStore.getPreflightSpecBySlug(watchSlug);

      if (!preflightSpec) {
        console.log('NOT FOUND!?');
        response.send(404);
        return;
      }
      const specString = preflightSpec.spec;
      const params = await Params.getParams();
      const putUrl = `https://${params.apiAdvertiseEndpoint}/api/v1/preflight/${watchSlug}/${clusterSlug}`

      const parsedPreflightSpec = jsYaml.load(specString);
      parsedPreflightSpec.spec.sendResultsTo = putUrl;
      console.log(parsedPreflightSpec);
      console.log('URL:', putUrl);
      response.send(200, parsedPreflightSpec);


    } catch (err) {
      throw err;
    }


  }

  @Put("/*")
  async putPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    // @PathParams("downstream_cluster") downstream_cluster: string
  ): Promise<void> {
    console.log('GO PUT REQUEST!!!!');
    // Write preflight results to the database
    // const result = request.body;
    // await request.app.locals.stores.preflightStore.addPreflightResult(watchId, result);
    response.send(200);
  }
}
