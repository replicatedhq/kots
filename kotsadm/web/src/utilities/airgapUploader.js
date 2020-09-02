import Resumable from "resumablejs";
import { Utilities } from "./utilities";

export class AirgapUploader {
  constructor(isUpdate, onFileAdded) {
    this.isUpdate = isUpdate;

    this.resumableUploader = new Resumable({
      target: `${window.env.API_ENDPOINT}/app/airgap/chunk`,
      headers: {
        "Authorization": Utilities.getToken(),
      },
      fileType: ["airgap"],
      maxFiles: 1,
      simultaneousUploads: 3,
      maxChunkRetries: 0,
    });
  
    this.resumableUploader.on('fileAdded', (resumableFile) => {
      this.attemptedFileUpload = false;
      this.resumableFile = resumableFile;
      this.resumableIdentifier = resumableFile.uniqueIdentifier;
      this.resumableTotalChunks = resumableFile.chunks.length;
      if (onFileAdded) {
        onFileAdded(resumableFile.file);
      }
    });
  }

  upload = async (processParams, onProgress, onError, onComplete) => {
    try {
      this.processParams = processParams;

      const bundleExists = await this.airgapBundleExists();
      if (bundleExists) {
        onProgress(1, this.resumableUploader.getSize()); // progress 1 => 100%
        await this.processAirgapBundle();
        if (onComplete) {
          onComplete();
        }
        return;
      }

      if (this.attemptedFileUpload) {
        this.resumableFile.retry();
        return;
      }

      this.resumableUploader.on('fileProgress', () => {
        const progress = this.resumableUploader.progress();
        const size = this.resumableUploader.getSize();
        if (onProgress) {
          onProgress(progress, size);
        }
      });

      this.resumableUploader.on('fileError', (_, message) => {
        if (onError) {
          const errMsg = message ? message : "Error uploading bundle, please try again";
          onError(errMsg);
        }
      });

      this.resumableUploader.on('fileSuccess', async () => {
        await this.processAirgapBundle();
        if (onComplete) {
          onComplete();
        }
      });

      this.resumableUploader.upload();
      this.attemptedFileUpload = true;
    } catch(err) {
      console.log(err);
      if (onError) {
        const errMsg = err ? err.message : "Error uploading bundle, please try again";
        onError(errMsg);
      }
    }
  }

  airgapBundleExists = async () => {
    const res = await fetch(`${window.env.API_ENDPOINT}/app/airgap/bundleexists/${this.resumableIdentifier}/${this.resumableTotalChunks}`, {
      headers: {
        "Authorization": Utilities.getToken(),
      },
      method: "GET",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.exists;
  }

  processAirgapBundle = async () => {
    const res = await fetch(`${window.env.API_ENDPOINT}/app/airgap/processbundle/${this.resumableIdentifier}/${this.resumableTotalChunks}`, {
      headers: {
        "Authorization": Utilities.getToken(),
      },
      body: JSON.stringify(this.processParams),
      method: this.isUpdate ? "PUT" : "POST",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  }

  assignElement = element => {
    this.resumableUploader.assignBrowse(element);
    this.resumableUploader.assignDrop(element);
  }
}