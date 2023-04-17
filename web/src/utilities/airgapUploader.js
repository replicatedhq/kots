// TODO: fix linting issues
/* eslint-disable */
import Resumable from "resumablejs";
import { Utilities } from "./utilities";
import fetch from "./fetchWithTimeout";

export class AirgapUploader {
  constructor(isUpdate, appSlug, onFileAdded, simultaneousUploads) {
    this.isUpdate = isUpdate;
    this.appSlug = appSlug;

    this.resumableUploader = new Resumable({
      target: `${process.env.API_ENDPOINT}/app/${this.appSlug}/airgap/chunk`,
      credentials: "include",
      fileType: ["airgap"],
      maxFiles: 1,
      simultaneousUploads,
      maxChunkRetries: 0,
      xhrTimeout: 10000,
    });

    this.resumableUploader.on("fileAdded", (resumableFile) => {
      this.attemptedFileUpload = false;
      this.resumableFile = resumableFile;
      this.resumableIdentifier = resumableFile.uniqueIdentifier;
      this.resumableTotalChunks = resumableFile.chunks.length;
      if (onFileAdded) {
        onFileAdded(resumableFile.file);
      }
    });
  }

  reconnect = async (reconnectAttempt = 0) => {
    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/ping`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
        },
        10000
      );

      if (res.status === 401) {
        Utilities.logoutUser();
        return false;
      }

      return true;
    } catch (_) {
      reconnectAttempt++;
      if (reconnectAttempt > 10) {
        return false;
      }
      const reconnectPromise = new Promise((resolve) => {
        setTimeout(() => {
          this.reconnect(reconnectAttempt).then(resolve);
        }, 1000);
      });
      return await reconnectPromise;
    }
  };

  upload = async (processParams, onProgress, onError, onComplete) => {
    try {
      // first, validate that the release is compatible with the current kots version.
      // don't block the upload process if compatibility couldn't be determined here
      // since this is just to fail early and the api will recheck for compatibility later on.
      let installableResponse;
      try {
        const appSpec = await Utilities.getAppSpecFromAirgapBundle(
          this.resumableFile.file
        );
        const airgapSpec = await Utilities.getAirgapMetaFromAirgapBundle(
          this.resumableFile.file
        );
        installableResponse = await this.canInstallRelease(appSpec, airgapSpec);
      } catch (err) {
        console.log(err);
      }

      if (installableResponse?.canInstall === false) {
        throw new Error(installableResponse?.error);
      }

      this.processParams = processParams;
      this.onProgress = onProgress;
      this.onError = onError;
      this.onComplete = onComplete;

      const bundleExists = await this.airgapBundleExists();
      if (bundleExists) {
        this.onProgress(1, this.resumableUploader.getSize()); // progress 1 => 100%
        await this.processAirgapBundle();
        if (onComplete) {
          this.onComplete();
        }
        return;
      }

      // set the initial progress to the current api progress
      this.apiCurrentProgress = await this.getApiCurrentProgress();
      if (this.onProgress) {
        const size = this.resumableUploader.getSize();
        this.onProgress(this.apiCurrentProgress, size);
      }

      if (this.attemptedFileUpload) {
        this.resumableFile.retry();
        return;
      }

      if (!this.hasListeners) {
        this.resumableUploader.on("fileProgress", () => {
          if (this.onProgress) {
            // the resumablejs library returns progress as 1 in both cases of "error" and "success"
            // we don't wanna show the progress as 100% while reconnecting in case of an error (upload is not complete)
            const progress = this.resumableUploader.progress();
            if (progress === 1 && !this.resumableFile.isComplete()) {
              return;
            }
            const size = this.resumableUploader.getSize();
            if (progress < this.apiCurrentProgress) {
              // when an error occurs during uploading one of the chunks, the uploader or the user will retry uploading the file from the
              // beginning to check if any previously uploaded chunks were lost. during that process, the progress will be less than the
              // actual progress if no data loss occured, so we keep the UI progress as is until it catches up, and show a "resuming" message to the user.
              this.onProgress(this.apiCurrentProgress, size, true);
              return;
            }
            this.onProgress(progress, size);
          }
        });

        this.resumableUploader.on("fileError", async (_, message) => {
          // an error occured while uploading one of the chunks due to internet connectivity issues or the api pod restarting.
          // try reconnecting to the api. if reconnected successfully, get the actual current progress from the api and retry uploading the file.
          // this also handles an issue where the api pod loses all data related to the bundle when restarted.
          const reconnected = await this.reconnect();
          if (reconnected) {
            this.apiCurrentProgress = await this.getApiCurrentProgress();
            this.resumableFile.retry();
            return;
          }
          if (this.onError) {
            const errMsg =
              message || "Error uploading bundle, please try again";
            this.onError(errMsg);
          }
        });

        this.resumableUploader.on("fileSuccess", async () => {
          await this.processAirgapBundle();
          if (this.onComplete) {
            this.onComplete();
          }
        });

        this.hasListeners = true;
      }

      this.resumableUploader.upload();
      this.attemptedFileUpload = true;
    } catch (err) {
      console.log(err);
      if (onError) {
        const errMsg = err
          ? err.message
          : "Error uploading bundle, please try again";
        onError(errMsg);
      }
    }
  };

  canInstallRelease = async (appSpec, airgapSpec) => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${this.appSlug}/can-install`,
      {
        credentials: "include",
        body: JSON.stringify({
          appSpec: appSpec || "",
          airgapSpec: airgapSpec || "",
          isInstall: !this.isUpdate,
        }),
        method: "POST",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response;
  };

  getApiCurrentProgress = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${this.appSlug}/airgap/bundleprogress/${this.resumableIdentifier}/${this.resumableTotalChunks}`,
      {
        credentials: "include",
        method: "GET",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.progress;
  };

  airgapBundleExists = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${this.appSlug}/airgap/bundleexists/${this.resumableIdentifier}/${this.resumableTotalChunks}`,
      {
        credentials: "include",
        method: "GET",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.exists;
  };

  processAirgapBundle = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${this.appSlug}/airgap/processbundle/${this.resumableIdentifier}/${this.resumableTotalChunks}`,
      {
        credentials: "include",
        body: JSON.stringify(this.processParams),
        method: this.isUpdate ? "PUT" : "POST",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  assignElement = (element) => {
    this.resumableUploader.assignBrowse(element);
    this.resumableUploader.assignDrop(element);
  };
}
