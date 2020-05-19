import { Parse as TarParser } from "tar";
import { PassThrough as PassThroughStream } from "stream";
import path from "path";
import * as _ from "lodash";
import * as tar from "tar-stream";
import * as zlib from "zlib";
import concat from "concat-stream";
import yaml from "js-yaml";

interface InstallationSpec {
  cursor: string;
  channelName: string;
  versionLabel: string;
  releaseNotes: string;
  encryptionKey: string;
}

function bufferToStream(buffer: Buffer): NodeJS.ReadableStream {
  const stream = new PassThroughStream();
  stream.end(buffer);
  return stream;
}

export function extractKotsAppSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let appSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "Application") {
          appSpec = data.toString();
          resolve(appSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(appSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractConfigSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let configSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "Config") {
          configSpec = data.toString();
          resolve(configSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(configSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractConfigValuesFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let configValues = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "ConfigValues") {
          configValues = data.toString();
          resolve(configValues);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(configValues);
    });

    extract.end(uncompressed);
  });
}

export function extractAppSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let appSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "app.k8s.io/v1beta1" && doc.kind === "Application") {
          appSpec = data.toString();
          resolve(appSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(appSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractPreflightSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let preflight = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "troubleshoot.replicated.com/v1beta1" && doc.kind === "Preflight") {
          preflight = data.toString();
          resolve(preflight);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(preflight);
    });

    extract.end(uncompressed);
  });
}

export function extractBackupSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "velero.io/v1" && doc.kind === "Backup") {
          resolve(data.toString());
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(null);
    });

    extract.end(uncompressed);
  });
}

export function extractInstallationSpecFromTarball(tarball: Buffer): Promise<InstallationSpec> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat((data) => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if ((doc.apiVersion === "kots.io/v1beta1") && (doc.kind === "Installation")) {
          const spec = {
            cursor: doc.spec.updateCursor,
            channelName: doc.spec.channelName || "",
            versionLabel: doc.spec.versionLabel || "Unknown",
            releaseNotes: doc.spec.releaseNotes || "",
            encryptionKey: doc.spec.encryptionKey || "",
          };

          resolve(spec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve({
        cursor: "",
        channelName: "",
        versionLabel: "",
        releaseNotes: "",
        encryptionKey: ""
      });
    });

    extract.end(uncompressed);
  });
}

export function extractRawInstallationSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let installationSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());
        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "Installation") {
          installationSpec = data.toString();
          resolve(installationSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(installationSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractDownstreamNamesFromTarball(tarball: Buffer): Promise<string[]> {
  return new Promise<string[]>((resolve, reject) => {
    let downstreamNames: string[] = [];
    const parser = new TarParser({
      strict: true,
      filter: (currentPath: string) => {
        const parts = currentPath.split(path.sep);
        _.remove(parts, (n) => {
          return n.length === 0;
        });

        // the first part is always the name of the directory it was uploaded from
        if (parts.length === 4) {
          if (parts[0] === "overlays" && parts[1] === "downstreams" && parts[3] === "kustomization.yaml") {
            downstreamNames.push(parts[2]);
          }
        }
        return false;
      },
    });
    bufferToStream(tarball)
      .pipe(parser)
      .on('end', () => {
        resolve(downstreamNames);
      })
      .on('error', reject);
  });
}

export function extractAnalyzerSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let analyzerSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());

        if (doc.apiVersion === "troubleshoot.replicated.com/v1beta1" && doc.kind === "Analyzer") {
          analyzerSpec = data.toString();
          resolve(analyzerSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(analyzerSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractSupportBundleSpecFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let bundleSpec = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());

        if (doc.apiVersion === "troubleshoot.replicated.com/v1beta1" && doc.kind === "Collector") {
          bundleSpec = data.toString();
          resolve(bundleSpec);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(bundleSpec);
    });

    extract.end(uncompressed);
  });
}

export function extractKotsAppLicenseFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let license = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());

        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "License") {
          license = data.toString();
          resolve(license);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(license);
    });

    extract.end(uncompressed);
  });
}

export function extractAppTitleFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let appTitle = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());

        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "Application") {
          appTitle = doc.spec.title;
          resolve(appTitle);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(appTitle);
    });

    extract.end(uncompressed);
  });
}

export function extractAppIconFromTarball(tarball: Buffer): Promise<string | null> {
  const uncompressed = zlib.unzipSync(tarball);
  const extract = tar.extract();

  let appIcon = null;

  return new Promise((resolve, reject) => {
    extract.on("error", reject);

    extract.on("entry", (header, stream, next) => {
      stream.pipe(concat(data => {
        if (!isYaml(data.toString())) {
          next();
          return;
        }

        const doc = yaml.safeLoad(data.toString());

        if (doc.apiVersion === "kots.io/v1beta1" && doc.kind === "Application") {
          appIcon = doc.spec.icon;
          resolve(appIcon);
          next();
          return;
        }
        next();
      }));
    });

    extract.on("finish", () => {
      resolve(appIcon);
    });

    extract.end(uncompressed);
  });
}

function isYaml(data: string): boolean {
  try {
    const doc = yaml.safeLoad(data.toString());
    if (doc && doc.apiVersion) {
      // we only support kubernetes yaml, so this is a little bit rough,
      // but should be valid for now
      return true;
    }
  } catch (err) {
    /* nothing */
  }

  return false;
}
