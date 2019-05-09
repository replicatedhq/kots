import { S3 } from "aws-sdk";
import * as Bluebird from "bluebird";
import * as Gunzip from "gunzip-maybe";
import * as jaeger from "jaeger-client";
import { instrumented } from "monkit";
import { Writable } from "stream";
import { extract, Extract } from "tar-stream";
import { logger } from "../server/logger";
import { tracer } from "../server/tracing";
import { getS3 } from "../util/s3";
import { WatchStore } from "./watch_store";
import { Params } from "../server/params";

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

  @instrumented()
  async downloadDeploymentYAML(watchId: string): Promise<DeploymentFile> {
    const span = tracer().startSpan("downloadDeploymentYAML");
    const watch = await this.watchStore.getWatch(span, watchId);
    const params = await this.watchStore.getLatestGeneratedFileS3Params(span, watchId);

    const download = await this.findDeploymentFile(span, params);
    const filename = this.determineFileName(download, watch.watchName!);
    span.finish();
    return {
      ...download,
      filename,
    };
  }

  @instrumented()
  async downloadDeploymentYAMLForSequence(watchId: string, sequence: number): Promise<DeploymentFile> {
    const span = tracer().startSpan("downloadDeploymentYAMLForSequence");
    const params = await this.watchStore.getLatestGeneratedFileS3Params(span.context(), watchId, sequence);

    const download = await this.findDeploymentFile(span.context(), params);
    const watch = await this.watchStore.getWatch(span.context(), watchId);
    const filename = this.determineFileName(download, watch.watchName!);
    span.finish();
    return {
      ...download,
      filename,
    };
  }

  @instrumented()
  async findDeploymentFile(ctx: jaeger.SpanContext, params: S3.Types.GetObjectRequest): Promise<Download> {
    const span: jaeger.SpanContext = tracer().startSpan("watchDownload.findDeploymentYAML", { childOf: ctx });
    const shipParams = await Params.getParams();

    return new Bluebird<Download>((resolve, reject) => {
      const s3 = getS3(shipParams);
      let yamlBuffer = new Buffer("");
      let tarGzBuffer = new Buffer("");

      const s3TarStream = s3.getObject(params).createReadStream();
      const gunzip: Writable = Gunzip();
      const extractor: Extract = extract();

      s3TarStream.on("error", reject);
      gunzip.on("error", reject);
      extractor.on("error", reject);
      extractor.on("pipe", () => {
        logger.debug("Tar reader stream started");
      });

      let deploymentYAMLFound = false;
      extractor.on("entry", ({ name }, stream, next) => {
        stream.on("error", reject);

        if ((name.endsWith("rendered.yaml")) || (name.endsWith("ship-enterprise.yaml"))) {
          logger.debug("Found rendered.yaml or ship-enterprise.yaml");
          deploymentYAMLFound = true;
          stream.on("data", (chunk: Buffer) => {
            yamlBuffer = Buffer.concat([yamlBuffer, chunk]);
          });
          stream.on("finish", () => {
            logger.debug("Streamed rendered.yaml or ship-enterprise.yaml to yamlBuffer");
            next();
          });
        } else {
          stream.on("end", next);
          stream.resume();
        }
      });

      extractor.on("finish", () => {
        logger.debug("Finished reading tar");

        let download: Download;
        if (deploymentYAMLFound) {
          download = {
            contents: yamlBuffer,
            contentType: ContentType.YAML,
          };
        } else {
          logger.debug("No deployment YAML found, falling back to tar");
          download = {
            contents: tarGzBuffer,
            contentType: ContentType.TarGZ,
          };
        }

        span.finish();
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
        span.finish();
        throw error;
      });
  }

  private determineFileName(download: Download, watchName: string): string {
    return `${watchName}${FileExtensions[download.contentType]}`;
  }
}
