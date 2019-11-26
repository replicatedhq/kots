import { Controller, Get, Put, Post, BodyParams, Req, Res, PathParams, HeaderParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";
import { Request, Response } from "express";
import { putObject, upload } from "../../util/s3";
import { Params } from "../../server/params";
import path from "path";
import fs from "fs";
import tmp from "tmp";
import * as _ from "lodash";
import {
  extractDownstreamNamesFromTarball,
  extractInstallationSpecFromTarball,
  extractPreflightSpecFromTarball,
  extractAppSpecFromTarball,
  extractKotsAppSpecFromTarball,
  extractSupportBundleSpecFromTarball,
  extractAppTitleFromTarball,
  extractAppIconFromTarball,
  extractKotsAppLicenseFromTarball,
  extractAnalyzerSpecFromTarball
} from "../../util/tar";
import { Cluster } from "../../cluster";
import { KotsApp, kotsAppFromLicenseData } from "../../kots_app";
import { extractFromTgzStream, getImageFiles, getImageFormats, pathToShortImageName, pathToImageName } from "../../airgap/archive";
import { StatusServer } from "../../airgap/status";
import {
  kotsPullFromAirgap,
  kotsAppFromAirgapData,
  kotsTestRegistryCredentials
} from "../../kots_app/kots_ffi";
import { Session } from "../../session";
import { getDiffSummary } from "../../util/utilities";
import yaml from "js-yaml";
import * as k8s from "@kubernetes/client-node";
import { decodeBase64 } from "../../util/utilities";
import { Repeater } from "../../util/repeater";
import { KotsAppStore } from "../../kots_app/kots_app_store";
import { createGitCommitForVersion } from "../../kots_app/gitops";

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
  @Get("/ports")
  async kotsPorts(
    @Req() request: Request,
    @Res() response: Response,
  ): Promise<any> {
    // This method is connected to over kubectl...
    // There is no user auth, but this method should be
    // exposed only on cluster ip to enforce that it's
    // not exposed to the public

    const kotsAppStore: KotsAppStore = request.app.locals.stores.kotsAppStore;

    const apps = await kotsAppStore.listInstalledKotsApps();
    if (apps.length === 0) {
      return [];
    }
    const app = apps[0];

    if (!app.currentSequence) {
      return [];
    }

    const appSpec = await kotsAppStore.getAppSpec(app.id, app.currentSequence);
    if (!appSpec) {
      return [];
    }

    const parsedKotsAppSpec = await kotsAppStore.getKotsAppSpec(app.id, app.currentSequence);
    try {
      const parsedAppSpec = yaml.safeLoad(appSpec);
      if (!parsedKotsAppSpec) {
        return [];
      }

      const ports: any[] = [];
      for (const link of parsedAppSpec.spec.descriptor.links) {
        if (parsedKotsAppSpec.ports) {
          const mapped = _.find(parsedKotsAppSpec.ports, (port: any) => {
            return port.applicationUrl === link.url;
          });

          if (mapped) {
            ports.push(mapped);
          }
        }
      }

      return ports;
    } catch (err) {
      console.log(err);
      return [];
    }
  }

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
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {

    // Intentionally not processing registry settings here because empty strings don't
    // necessarily mean existing info should be deleted.

    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

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
    @Res() response: Response,
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {

    // kots install command is allowed to install the first app without auth.
    const apps = await request.app.locals.stores.kotsAppStore.listInstalledKotsApps();
    if (apps.length > 0) {
      if (!auth) {
        response.status(401);
        return {};
      }

      const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
      if (!session || !session.userId) {
        response.status(401);
        return {};
      }
    }

    const metadata = JSON.parse(body.metadata);

    const kotsApp = await request.app.locals.stores.kotsAppStore.createKotsApp(metadata.name, metadata.upstreamURI, metadata.license);
    await request.app.locals.stores.kotsAppStore.updateRegistryDetails(kotsApp.id, metadata.registryEndpoint, metadata.registryUsername, metadata.registryPassword, metadata.registryNamespace);

    const params = await Params.getParams();
    const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, "0.tar.gz");
    const buffer = fs.readFileSync(file.path);
    await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

    const installationSpec = await extractInstallationSpecFromTarball(buffer);
    const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
    const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
    const preflightSpec = await extractPreflightSpecFromTarball(buffer);
    const appSpec = await extractAppSpecFromTarball(buffer);
    const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
    const appTitle = await extractAppTitleFromTarball(buffer);
    const appIcon = await extractAppIconFromTarball(buffer);
    const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);

    await request.app.locals.stores.kotsAppStore.createMidstreamVersion(
      kotsApp.id,
      0,
      installationSpec.versionLabel,
      installationSpec.releaseNotes,
      installationSpec.cursor,
      installationSpec.encryptionKey,
      supportBundleSpec,
      analyzersSpec,
      preflightSpec,
      appSpec,
      kotsAppSpec,
      kotsAppLicense,
      appTitle,
      appIcon
    );

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
      await request.app.locals.stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, installationSpec.versionLabel, "deployed", "Kots Install", "", "");
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

    return uploadUpdate(stores, metadata.slug, buffer, "Kots Upload");
  }

  @Post("/airgap")
  async kotsUploadAirgap(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: any,
    @Req() request: Request,
    @Res() response: Response,
  ): Promise<any> {
    const params = await Params.getParams();

    const app = await request.app.locals.stores.kotsAppStore.getPendingKotsAirgapApp();

    let registryHost, namespace, username, password = "";

    let needsRegistry = true;
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
      const res = await k8sApi.readNamespacedSecret("registry-creds", "default");
      if (res && res.body && res.body.data && res.body.data[".dockerconfigjson"]) {
        needsRegistry = false;

        // parse the dockerconfig secret
        const parsed = JSON.parse(decodeBase64(res.body.data[".dockerconfigjson"]));
        const auths = parsed.auths;
        for (const hostname in auths) {
          const config = auths[hostname];
          if (config.username === "kurl") {
            registryHost = hostname;
            username = config.username;
            password = config.password;
            namespace = app.slug;
          }
        }
      }
    } catch {
      /* no need to handle, rbac problem or not a path we can read registry */
    }

    if (needsRegistry) {
      registryHost = body.registryHost;
      namespace = body.namespace;
      username = body.username;
      password = body.password;
    }

    const liveness = new Repeater(() => {
      return new Promise((resolve) => {
        request.app.locals.stores.kotsAppStore.updateAirgapInstallLiveness().finally(() => {
          resolve();
        })
      });
    }, 1000);

    const dstDir = tmp.dirSync();
    var appSlug: string;
    let hasPreflight: Boolean;
    let isConfigurable: Boolean;
    try {
      await request.app.locals.stores.kotsAppStore.updateRegistryDetails(app.id, registryHost, username, password, namespace);
      await request.app.locals.stores.kotsAppStore.resetAirgapInstallInProgress(app.id);

      liveness.start();

      await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Processing package...", "running");
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

      const clusters = await request.app.locals.stores.clusterStore.listAllUsersClusters();
      let downstream;
      for (const cluster of clusters) {
        if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
          downstream = cluster;
        }
      }

      const tmpDstDir = tmp.dirSync();
      try {
        await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Processing app package...", "running");

        const out = path.join(tmpDstDir.name, "archive.tar.gz");

        const statusServer = new StatusServer();
        await statusServer.start(dstDir.name);
        const args = kotsPullFromAirgap(statusServer.socketFilename, out, app, String(app.license), dstDir.name, downstream.title, request.app.locals.stores, registryHost, namespace, username, password);
        await statusServer.connection();
        await statusServer.termination((resolve, reject, obj): boolean => {
          // Return true if completed
          if (obj.status === "running") {
            request.app.locals.stores.kotsAppStore.setAirgapInstallStatus(obj.display_message, "running");
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

        await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus("Creating app...", "running");
        const appProps = await kotsAppFromAirgapData(out, app, request.app.locals.stores);
        hasPreflight = appProps.hasPreflight;
        isConfigurable = appProps.isConfigurable;
      } finally {
        tmpDstDir.removeCallback();
      }

      appSlug = app.slug;
      await request.app.locals.stores.kotsAppStore.clearAirgapInstallInProgress();

    } catch(err) {

      await request.app.locals.stores.kotsAppStore.setAirgapInstallStatus(String(err), "failed");
      await request.app.locals.stores.kotsAppStore.setAirgapInstallFailed(app.id);
      throw(err);

    } finally {
      liveness.stop();
      dstDir.removeCallback();
    }

    response.header("Content-Type", "application/json");
    response.status(200);
    return {
      slug: appSlug,
      hasPreflight,
      isConfigurable,
    };
  }

  @Post("/airgap/reset/:slug")
  async kotsResetAirgapUpload(
    @Req() request: Request,
    @Res() response: Response,
    @HeaderParams("Authorization") auth: string,
  ) {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

    const slug = request.params.slug;

    const appId = await request.app.locals.stores.kotsAppStore.getIdFromSlug(slug);
    await request.app.locals.stores.kotsAppStore.resetAirgapInstallInProgress(appId);
    response.send(200);
  }

  @Post("/registry")
  async kotsValidateRegistryAuth(
    @BodyParams("") body: any,
    @Req() request: Request,
    @Res() response: Response,
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

    const { registryHost, namespace, username, password } = body;

    const testError = await kotsTestRegistryCredentials(registryHost, username, password, namespace);

    if (!testError) {
      response.send(200);
    } else {
      response.status(401);
    }
    return {error: testError};
  }
}

export async function uploadUpdate(stores, slug, buffer, source) {
  // Todo this could use some proper not-found error handling stuffs
  const kotsApp = await stores.kotsAppStore.getApp(await stores.kotsAppStore.getIdFromSlug(slug));

  const newSequence = kotsApp.currentSequence + 1;

  const params = await Params.getParams();
  const objectStorePath = path.join(params.shipOutputBucket.trim(), kotsApp.id, `${newSequence}.tar.gz`);
  await putObject(params, objectStorePath, buffer, params.shipOutputBucket);

  const supportBundleSpec = await extractSupportBundleSpecFromTarball(buffer);
  const analyzersSpec = await extractAnalyzerSpecFromTarball(buffer);
  const preflightSpec = await extractPreflightSpecFromTarball(buffer);
  const appSpec = await extractAppSpecFromTarball(buffer);
  const kotsAppSpec = await extractKotsAppSpecFromTarball(buffer);
  const appTitle = await extractAppTitleFromTarball(buffer);
  const appIcon = await extractAppIconFromTarball(buffer);
  const installationSpec = await extractInstallationSpecFromTarball(buffer);
  const kotsAppLicense = await extractKotsAppLicenseFromTarball(buffer);

  await stores.kotsAppStore.createMidstreamVersion(
    kotsApp.id,
    newSequence,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    kotsAppLicense,
    appTitle,
    appIcon
  );

  const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(kotsApp.id);
  for (const clusterId of clusterIds) {
    const downstreamGitops = await stores.kotsAppStore.getDownstreamGitOps(kotsApp.id, clusterId);

    let commitUrl = "";
    if (downstreamGitops.enabled) {
      const commitMessage = `${source} for ${kotsApp.name}`;
      commitUrl = await createGitCommitForVersion(stores, kotsApp.id, clusterId, newSequence, commitMessage);
    }

    const status = preflightSpec
      ? "pending_preflight"
      : "pending";
    const diffSummary = await getDiffSummary(kotsApp);
    await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, newSequence, clusterId, installationSpec.versionLabel, status, source, diffSummary, commitUrl);
  }

  return {
    uri: `${params.shipApiEndpoint}/app/${kotsApp.slug}`,
  };
}
