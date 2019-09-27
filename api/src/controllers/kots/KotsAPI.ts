import { Controller, Get, Put, Post, BodyParams, Req, Res, PathParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";
import { Request, Response } from "express";
import { putObject, upload } from "../../util/s3";
import { Params } from "../../server/params";
import path from "path";
import fs from "fs";
import tmp from "tmp";
import * as _ from "lodash";
import { extractDownstreamNamesFromTarball, extractCursorAndVersionFromTarball } from "../../util/tar";
import { Cluster } from "../../cluster";
import { KotsApp, kotsAppFromLicenseData } from "../../kots_app";
import { extractFromTgzStream, getImageFiles, getImageFormats, pathToShortImageName, pathToImageName } from "../../airgap/archive";
import { StatusServer } from "../../airgap/status";
import { kotsPullFromAirgap, kotsAppFromAirgapData, kotsRewriteAndPushImageName } from "../../kots_app/kots_ffi";

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
  //   const apps = await request.app.locals.stores.kotsAppStore.listInstalledKotsApps();

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

    const apps = await request.app.locals.stores.kotsAppStore.listInstalledKotsApps();
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

    const cursorAndVersion = await extractCursorAndVersionFromTarball(buffer);
    await request.app.locals.stores.kotsAppStore.createMidstreamVersion(kotsApp.id, 0, cursorAndVersion.versionLabel, cursorAndVersion.cursor, undefined, undefined);

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
      await request.app.locals.stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, cursorAndVersion.versionLabel, "deployed");
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
    const buffer = fs.readFileSync(file.path);
    const stores = request.app.locals.stores;

    return uploadUpdate(stores, metadata.slug, buffer);
  }

  @Post("/airgap")
  async kotsUploadAirgap(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: any,
    @Req() request: Request,
    @Res() response: Response,
  ): Promise<any> {
    const { registryHost, namespace, username, password } = body;

    const params = await Params.getParams();

    const app = await request.app.locals.stores.kotsAppStore.getPendingKotsAirgapApp();

    const dstDir = tmp.dirSync();
    var appSlug: string;
    let hasPreflight: Boolean;
    var liveness: any;
    try {
      await request.app.locals.stores.kotsAppStore.setAirgapInstallInProgress(app.id);

      liveness = setInterval(() => {
        Promise.all([request.app.locals.stores.kotsAppStore.updateAirgapInstallLiveness()]);
      }, 1000);

      await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Uploading...");
      await upload(params, file.originalname, fs.createReadStream(file.path), params.airgapBucket);

      await extractFromTgzStream(fs.createReadStream(file.path), dstDir.name);
      const imagesRoot = path.join(dstDir.name, "images");

      const imageFormats = getImageFormats(imagesRoot);

      let imageMap: any[] = [];
      for (const format of imageFormats) {
        const formatRoot = path.join(imagesRoot, format);
        const files = await getImageFiles(formatRoot);

        const m = files.map(imageFile => {
          return {
            format: format,
            filePath: imageFile,
            shortName: pathToShortImageName(formatRoot, imageFile),
            fullName: pathToImageName(formatRoot, imageFile),
          }
        });
        imageMap = imageMap.concat(m);
      }

      for (const image of imageMap) {
        const statusServer = new StatusServer();
        await statusServer.start(dstDir.name);
        const args = kotsRewriteAndPushImageName(statusServer.socketFilename, image.filePath, image.shortName, image.format, registryHost, namespace, username, password);
        await statusServer.connection();
        await statusServer.termination((resolve, reject, obj): boolean => {
          // Return true if completed
          if (obj.status === "running") {
            Promise.all([request.app.locals.stores.kotsAppStore.setAirgapInstallStatus(obj.display_message)]);
            return false;
          } else if (obj.status === "terminated") {
            if (obj.exit_code === 0) {
              resolve();
            } else {
              reject(new Error(`process failed: ${obj.display_message}`));
            }
            return true;
          }
          return false;
        });
      }

      const clusters = await request.app.locals.stores.clusterStore.listAllUsersClusters();
      let downstream;
      for (const cluster of clusters) {
        if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
          downstream = cluster;
        }
      }

      const tmpDstDir = tmp.dirSync();
      try {
        await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Processing app package...");

        const out = path.join(tmpDstDir.name, "archive.tar.gz");

        const statusServer = new StatusServer();
        await statusServer.start(dstDir.name);
        const args = kotsPullFromAirgap(statusServer.socketFilename, out, app, String(app.license), dstDir.name, downstream.title, request.app.locals.stores, registryHost, namespace);
        await statusServer.connection();
        await statusServer.termination((resolve, reject, obj): boolean => {
          // Return true if completed
          if (obj.status === "terminated") {
            if (obj.exit_code === 0) {
              resolve();
            } else {
              reject(new Error(`process failed: ${obj.display_message}`));
            }
            return true;
          }
          return false;
        });

        await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Creating app...");
        const appProps = await kotsAppFromAirgapData(out, app, request.app.locals.stores);
        hasPreflight = appProps.hasPreflight;
      } finally {
        tmpDstDir.removeCallback();
      }

      await request.app.locals.stores.kotsAppStore.updateRegistryDetails(app.id, registryHost, username, password, namespace);

      appSlug = app.slug;
    } catch(err) {

      await request.app.locals.stores.kotsAppStore.setAirgapInstallFailed(app.id);
      throw(err);

    } finally {
      clearInterval(liveness);
      dstDir.removeCallback();
    }

    response.header("Content-Type", "application/json");
    response.status(200);
    return {
      slug: appSlug,
      hasPreflight: hasPreflight,
    };
  }
}

export async function uploadUpdate(stores, slug, buffer) {
  // Todo this could use some proper not-found error handling stuffs
  const kotsApp = await stores.kotsAppStore.getApp(await stores.kotsAppStore.getIdFromSlug(slug));

  const newSequence = kotsApp.currentSequence + 1;

  const params = await Params.getParams();
  const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, `${newSequence}.tar.gz`);
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const supportBundleSpec = undefined;
  const preflightSpec = undefined;

  const cursorAndVersion = await extractCursorAndVersionFromTarball(buffer);
  await stores.kotsAppStore.createMidstreamVersion(kotsApp.id, newSequence, cursorAndVersion.versionLabel, cursorAndVersion.cursor, undefined, undefined);

  const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(kotsApp.id);
  for (const clusterId of clusterIds) {
    await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, newSequence, clusterId, cursorAndVersion.versionLabel, "pending");
  }

  return {
    uri: `${params.shipApiEndpoint}/app/${kotsApp.slug}`,
  };
}
