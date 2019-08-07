import { Request, Response } from "express";
import { Controller, Get, Post, Res, Req } from "ts-express-decorators";
import jsYaml from "js-yaml";

import { Params } from "../server/params";

@Controller("/api/v1/preflight")
export class PreflightAPI {
  @Get("/*")
  async getPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,

  ): Promise<void> {
    const splitPath = request.path.split('/').slice(1);
    let watchSlug;
    let clusterSlug;

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

    try {
      // Fetch YAML from the database and return to client with injected key
      const preflightSpec = await request.app.locals.stores.preflightStore.getPreflightSpecBySlug(watchSlug);

      if (!preflightSpec) {
        console.log(`Preflight spec for slug: ${watchSlug} not found`);
        response.send(404);
        return;
      }
      const specString = preflightSpec.spec;
      const params = await Params.getParams();
      const putUrl = `${params.apiAdvertiseEndpoint}/api/v1/preflight/${watchSlug}/${clusterSlug}`;
      const parsedPreflightSpec = jsYaml.load(specString);
      parsedPreflightSpec.spec.uploadResultsTo = putUrl;

      response.send(200, parsedPreflightSpec);


    } catch (err) {
      throw err;
    }

  }

  @Post("/*")
  async putPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
  ): Promise<void> {

    const splitPath = request.path.split('/').slice(1);
    let watchSlug;

    if (splitPath.length === 3) {
      watchSlug = splitPath.slice(0, 2).join('/');

    } else if (splitPath.length === 2) {
      watchSlug = splitPath[0];

    } else {
      response.send(400);
      return;
    }

    // Write preflight results to the database
    const result = request.body;
    await request.app.locals.stores.preflightStore.addPreflightResult(watchSlug, result);
    response.send(200);
  }
}
