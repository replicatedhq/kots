import { Controller, Get, Put, Post, BodyParams, Req, Res, PathParams, HeaderParams, QueryParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";
import { Request, Response } from "express";
import { putObject } from "../../util/s3";
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
  extractAnalyzerSpecFromTarball,
  extractConfigSpecFromTarball,
  extractConfigValuesFromTarball,
  extractBackupSpecFromTarball
} from "../../util/tar";
import { Cluster } from "../../cluster";
import { KotsApp } from "../../kots_app";
import { StatusServer } from "../../airgap/status";
import {
  kotsTestRegistryCredentials,
  Update,
  kotsAppDownloadUpdateFromAirgap,
} from "../../kots_app/kots_ffi";
import { Session } from "../../session";
import { getDiffSummary } from "../../util/utilities";
import yaml from "js-yaml";
import { Repeater } from "../../util/repeater";
import { KotsAppStore } from "../../kots_app/kots_app_store";
import { createGitCommitForVersion } from "../../kots_app/gitops";

interface CreateAppBody {
  metadata: string;
}

interface UploadLicenseBody {
  name: string;
  license: string;
  appSlug: string;
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
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

    const kotsAppStore: KotsAppStore = request.app.locals.stores.kotsAppStore;

    const apps = await kotsAppStore.listInstalledKotsApps();
    if (apps.length === 0) {
      return [];
    }
    const app = apps[0];

    if (_.isUndefined(app.currentSequence)) {
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
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

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
    response.send(await app.getArchive('' + app.currentSequence));
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
    const configSpec = await extractConfigSpecFromTarball(buffer);
    const configValues = await extractConfigValuesFromTarball(buffer);
    const backupSpec = await extractBackupSpecFromTarball(buffer);

    await request.app.locals.stores.kotsAppStore.createMidstreamVersion(
      kotsApp.id,
      0,
      installationSpec.versionLabel,
      installationSpec.releaseNotes,
      installationSpec.cursor,
      installationSpec.channelName,
      installationSpec.encryptionKey,
      supportBundleSpec,
      analyzersSpec,
      preflightSpec,
      appSpec,
      kotsAppSpec,
      kotsAppLicense,
      configSpec,
      configValues,
      appTitle,
      appIcon,
      backupSpec
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
      await request.app.locals.stores.kotsAppStore.createDownstreamVersion(kotsApp.id, 0, cluster.id, installationSpec.versionLabel, "deployed", "Kots Install", "", "", false);
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
    @Res() response: Response,
    @HeaderParams("Authorization") auth: string,
  ): Promise<any> {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

    const metadata = JSON.parse(body.metadata);
    const buffer = fs.readFileSync(file.path);
    const stores = request.app.locals.stores;

    return uploadUpdate(stores, metadata.slug, buffer, "Kots Upload");
  }

  @Post("/airgap/update")
  async kotsUploadAirgapUpdate(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: any,
    @Req() request: Request,
    @Res() response: Response,
    @HeaderParams("Authorization") auth: string,
  ) {
    const session: Session = await request.app.locals.stores.sessionStore.decode(auth);
    if (!session || !session.userId) {
      response.status(401);
      return {};
    }

    const stores = request.app.locals.stores;

    const liveness = new Repeater(() => {
      return new Promise((resolve) => {
        request.app.locals.stores.kotsAppStore.updateUpdateDownloadStatusLiveness().finally(() => {
          resolve();
        })
      });
    }, 1000);

    // we are doing this asyncronously....
    const processFile = async () => {
      try {
        liveness.start();
        await stores.kotsAppStore.setUpdateDownloadStatus("Processing package...", "running");

        const app = await stores.kotsAppStore.getApp(body.appId);
        const registryInfo = await stores.kotsAppStore.getAppRegistryDetails(app.id);

        await kotsAppDownloadUpdateFromAirgap(file.path, app, registryInfo, stores);

        await request.app.locals.stores.kotsAppStore.clearUpdateDownloadStatus();

      } catch (err) {

        await request.app.locals.stores.kotsAppStore.setUpdateDownloadStatus(String(err), "failed");
        throw (err);

      } finally {
        liveness.stop();
      }
    }

    // tslint:disable-next-line no-floating-promises
    processFile();

    response.status(202);
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
    return { error: testError };
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
  const configSpec = await extractConfigSpecFromTarball(buffer);
  const configValues = await extractConfigValuesFromTarball(buffer);
  const backupSpec = await extractBackupSpecFromTarball(buffer);

  await stores.kotsAppStore.createMidstreamVersion(
    kotsApp.id,
    newSequence,
    installationSpec.versionLabel,
    installationSpec.releaseNotes,
    installationSpec.cursor,
    installationSpec.channelName,
    installationSpec.encryptionKey,
    supportBundleSpec,
    analyzersSpec,
    preflightSpec,
    appSpec,
    kotsAppSpec,
    kotsAppLicense,
    configSpec,
    configValues,
    appTitle,
    appIcon,
    backupSpec
  );

  const clusterIds = await stores.kotsAppStore.listClusterIDsForApp(kotsApp.id);
  for (const clusterId of clusterIds) {
    const downstreamGitops = await stores.kotsAppStore.getDownstreamGitOps(kotsApp.id, clusterId);

    let commitUrl = "";
    let gitDeployable = false;
    if (downstreamGitops.enabled) {
      const commitMessage = `${source} for ${kotsApp.name}`;
      commitUrl = await createGitCommitForVersion(stores, kotsApp.id, clusterId, newSequence, commitMessage);
      if (commitUrl !== "") {
        gitDeployable = true;
      }
    }

    const status = preflightSpec
      ? "pending_preflight"
      : "pending";
    const diffSummary = await getDiffSummary(kotsApp);
    await stores.kotsAppStore.createDownstreamVersion(kotsApp.id, newSequence, clusterId, installationSpec.versionLabel, status, source, diffSummary, commitUrl, gitDeployable);
  }

  return {
    uri: `${params.shipApiEndpoint}/app/${kotsApp.slug}`,
  };
}
