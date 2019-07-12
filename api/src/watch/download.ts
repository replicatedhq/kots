import { S3 } from "aws-sdk";
import * as Bluebird from "bluebird";
import * as Gunzip from "gunzip-maybe";
import { Writable } from "stream";
import { extract, Extract } from "tar-stream";
import { logger } from "../server/logger";
import { getS3 } from "../util/s3";
import { WatchStore } from "./watch_store";
import { Params } from "../server/params";
import { Watch } from "./watch";

export enum ContentType {
  TarGZ = "application/gzip",
  YAML = "text/yaml",
}

export interface Download {
  contentType: ContentType;
  contents: Buffer;
}

export interface DeploymentFile extends Download {
  filename: string;
}

const FileExtensions = {
  [ContentType.TarGZ]: ".tar.gz",
  [ContentType.YAML]: ".yaml",
};

export class WatchDownload {
  constructor(private readonly watchStore: WatchStore) {}

  async downloadDeploymentYAML(watch: Watch): Promise<DeploymentFile> {
    const params = await this.watchStore.getLatestGeneratedFileS3Params(watch.id);
    const download = await this.findDeploymentFile(params);
    const filename = this.determineFileName(download, watch.watchName);
    return {
      ...download,
      filename,
    };
  }

  async downloadDeploymentYAMLForSequence(watch: Watch, sequence: number): Promise<DeploymentFile> {
    const params = await this.watchStore.getLatestGeneratedFileS3Params(watch.id, sequence);

    const download = await this.findDeploymentFile(params);
    const filename = this.determineFileName(download, watch.watchName);
    return {
      ...download,
      filename,
    };
  }

  async findDeploymentFile(params: S3.Types.GetObjectRequest): Promise<Download> {
    const shipParams = await Params.getParams();

    return new Bluebird<Download>((resolve, reject) => {
      const s3 = getS3(shipParams);
      let yamlBuffer = new Buffer("");
      let otherRootYamlBuffer = new Buffer("");
      let tarGzBuffer = new Buffer("");

      const s3TarStream = s3.getObject(params).createReadStream();
      const gunzip: Writable = Gunzip();
      const extractor: Extract = extract();

      s3TarStream.on("error", reject);
      gunzip.on("error", reject);
      extractor.on("error", reject);
      extractor.on("pipe", () => {
        logger.debug({msg: "Tar reader stream started"});
      });

      let deploymentYAMLFound = false;
      let otherRootYamlFound = false;
      const otherRootYamlRegexp = new RegExp(`out\/(?!kustomization\.yaml)[0-9a-zA-Z\-_]+\.yaml$`); // match yaml files in the root of 'out' besides kustomization.yaml
      extractor.on("entry", ({ name }, stream, next) => {
        stream.on("error", reject);

        if ((name.endsWith("rendered.yaml"))) {
          logger.debug({msg: "Found rendered.yaml or ship-enterprise.yaml"});
          deploymentYAMLFound = true;
          stream.on("data", (chunk: Buffer) => {
            yamlBuffer = Buffer.concat([yamlBuffer, chunk]);
          });
          stream.on("finish", () => {
            logger.debug({msg: "Streamed rendered.yaml or ship-enterprise.yaml to yamlBuffer"});
            next();
          });
        } else if (otherRootYamlRegexp.test(name)) {
          logger.debug({msg: `Found root yaml candidate ${name}`});
          otherRootYamlFound = true;
          otherRootYamlBuffer = new Buffer("");
          stream.on("data", (chunk: Buffer) => {
            otherRootYamlBuffer = Buffer.concat([otherRootYamlBuffer, chunk]);
          });
          stream.on("finish", () => {
            logger.debug({msg: `Streamed ${name} to otherRootYamlBuffer`});
            next();
          });
        } else {
          stream.on("end", next);
          stream.resume();
        }
      });

      extractor.on("finish", () => {
        logger.debug({msg: "Finished reading tar"});

        let download: Download;
        if (deploymentYAMLFound) {
          download = {
            contents: yamlBuffer,
            contentType: ContentType.YAML,
          };
        } else if (otherRootYamlFound) {
          download = {
            contents: otherRootYamlBuffer,
            contentType: ContentType.YAML,
          };
        } else {
          logger.debug({msg: "No deployment YAML found, falling back to tar"});
          download = {
            contents: tarGzBuffer,
            contentType: ContentType.TarGZ,
          };
        }

        resolve(download);
        return;
      });

      s3TarStream.on("data", (chunk: Buffer) => {
        tarGzBuffer = Buffer.concat([tarGzBuffer, chunk]);
      });
      s3TarStream.pipe(gunzip).pipe(extractor);
    })
      .timeout(30000, "Unable to read deployment YAML from tar output")
      .catch(error => {
        throw error;
      });
  }

  private determineFileName(download: Download, watchName: string): string {
    return `${watchName}${FileExtensions[download.contentType]}`;
  }
}
