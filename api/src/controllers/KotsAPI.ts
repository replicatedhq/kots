import { Controller, Get, Put, Post, BodyParams, Req, Res, PathParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";
import { Request, Response } from "express";
import { putObject } from "../util/s3";
import { Params } from "../server/params";
import path from "path";
import fs from "fs";
import * as _ from "lodash";
import { extractDownstreamNamesFromTarball, extractCursorFromTarball } from "../util/tar";
import { Cluster } from "../cluster";
import { KotsApp, kotsAppFromLicenseData } from "../kots_app";

interface CreateAppBody {
  metadata: string;
}

interface CreateAppMetadata {
  name: string;
  versionLabel: string;
  upstreamURI: string;
  updateCursor: string;
  license: string;
}

interface UploadLicenseBody {
  name: string;
  license: string;
}

interface UpdateAppBody {
  slug: string;
}

@Controller("/api/v1/kots")
export class KotsAPI {
  // @Get("/apps")
  // async kotsList(
  //   @Req() request: Request,
  // ): Promise<any> {
  //   const apps = await request.app.locals.stores.kotsAppStore.listKotsApps();

  //   const result = _.map(apps, (app: KotsApp) => {
  //     return {
  //       name: app.name,
  //       slug: app.slug,
  //     };
  //   });

  //   return result;
  // }

  @Get("/:slug")
  async kotsDownload(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("slug") slug: string,
  ): Promise<any> {
    // this assumes single tenant and single app for now

    const apps = await request.app.locals.stores.kotsAppStore.listKotsApps();
    const app = _.find(apps, (a: KotsApp) => {
      return a.slug === slug;
    });

    if (!app) {
      response.status(404);
      return {};
    }

    response.header("Content-Type", "application/gzip");
    response.status(200);
    response.send(await app.getArchive(''+app.currentSequence));
  }

  @Post("/license")
  async kotsUploadLicense(
    @BodyParams("") body: UploadLicenseBody,
    @Req() request: Request,
    @Res() response: Response,
  ): Promise<any> {
    const clusters = await request.app.locals.stores.clusterStore.listAllUsersClusters();
    let downstream;
    for (const cluster of clusters) {
      if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
        downstream = cluster;
      }
    }

    const kotsApp = await kotsAppFromLicenseData(body.license, body.name, downstream.title, request.app.locals.stores);
    if (!kotsApp) {
      response.status(500);
      return {
        error: "failed to create app",
      };
    }

    const params = await Params.getParams();
    response.status(201);
    return {
      uri: `${params.shipApiEndpoint}/app/${kotsApp!.slug}`,
    };
  }

  @Post("/")
  async kotsUploadCreate(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: CreateAppBody,
    @Req() request: Request,
  ): Promise<any> {
    // IMPORTANT: this function does not have user-auth and is only usable in
    // single tenant (on-prem mode right now)

    const metadata = JSON.parse(body.metadata);

    const kotsApp = await request.app.locals.stores.kotsAppStore.createKotsApp(metadata.name, metadata.upstreamURI, metadata.license);

    const params = await Params.getParams();
    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
    const buffer = fs.readFileSync(file.path);
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    await request.app.locals.stores.kotsAppStore.createMidstreamVersion(kotsApp.id, 0, "??", await extractCursorFromTarball(buffer), undefined, undefined);

    // TODO parse and get support bundle and prefight from the upload
    const supportBundleSpec = undefined;
    const preflightSpec = undefined;

    // we have a local copy of the file now, let's look for downstreams
    const downstreams = await extractDownstreamNamesFromTarball(buffer);
    const clusters = await request.app.locals.stores.clusterStore.listAllUsersClusters();
    for (const downstream of downstreams) {
      const cluster = _.find(clusters, (c: Cluster) => {
        return c.title === downstream;
      });

      if (!cluster) {
        continue;
      }

      await request.app.locals.stores.kotsAppStore.createDownstream(kotsApp.id, downstream, cluster.id);
      await request.app.locals.stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id);
    }


    return {
      uri: `${params.shipApiEndpoint}/app/${kotsApp.slug}`,
    };
  }

  @Put("/")
  async kotsUploadUpdate(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: CreateAppBody,
    @Req() request: Request,
  ): Promise<any> {
    const metadata = JSON.parse(body.metadata);

    // Todo this could use some proper not-found error handling stuffs
    const kotsApp = await request.app.locals.stores.kotsAppStore.getApp(await request.app.locals.stores.kotsAppStore.getIdFromSlug(metadata.slug));

    const newSequence = kotsApp.currentSequence + 1;

    const params = await Params.getParams();
    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, `${newSequence}.tar.gz`);
    const buffer = fs.readFileSync(file.path);
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const supportBundleSpec = undefined;
    const preflightSpec = undefined;

    await request.app.locals.stores.kotsAppStore.createMidstreamVersion(kotsApp.id, newSequence, "??", await extractCursorFromTarball(buffer), undefined, undefined);

    const clusterIds = await request.app.locals.stores.kotsAppStore.listClusterIDsForApp(kotsApp.id);
    for (const clusterId of clusterIds) {
      await request.app.locals.stores.kotsAppStore.createDownstreamVersion(kotsApp.id, newSequence, clusterId);
    }

    return {
      uri: `${params.shipApiEndpoint}/app/${kotsApp.slug}`,
    };
  }
}
