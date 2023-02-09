import { useState, useEffect } from "react";
import { Utilities } from "../../../utilities/utilities";

const getValues = async ({
  _fetch = fetch,
  _token = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
  sequence,
  versionLabel,
  isPending,
}) => {
  try {
    const response = await _fetch(
      `${apiEndpoint}/app/${appSlug}/values/${sequence}?isPending=${isPending}&semver=${versionLabel}`,
      {
        method: "GET",
        headers: {
          Authorization: _token,
          "Content-Type": "application/blob",
        },
      }
    );
    if (!response.ok) {
      throw new Error("Error fetching values");
    }

    const data = await response.blob();
    return { data };
  } catch (error) {
    throw Error(error);
  }
};

const useDownloadValues = ({
  _createObjectURL = URL.createObjectURL,
  _getValues = getValues,
  _revokeObjectURL = URL.revokeObjectURL,
  appSlug,
  fileName,
  sequence,
  versionLabel,
  isPending,
} = {}) => {
  const [isDownloading, setIsDownloading] = useState(false);
  const [error, setError] = useState(null);
  const [url, setUrl] = useState(null);
  const [name, setName] = useState(null);

  // creates a download url and adds it to the dom triggering download of file defined in url
  useEffect(() => {
    if (url) {
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", name);

      document.body.appendChild(link);

      link.click();
      link.parentNode.removeChild(link);
      _revokeObjectURL(url);
      setUrl(null);
    }
  }, [url]);

  const download = async () => {
    try {
      setIsDownloading(true);
      setError(null);
      // TODO: error will never be returned. probably refactor to return error or use react-query
      const { data, error: _error } = await _getValues({
        appSlug,
        sequence,
        versionLabel,
        isPending,
      });
      if (_error) {
        setError(_error);
        setIsDownloading(false);
        return;
      }

      const newUrl = _createObjectURL(new Blob([data]));
      setUrl(newUrl);
      setName(fileName);
      setIsDownloading(false);
    } catch (downloadError) {
      setIsDownloading(false);
      setError(downloadError);
    }
  };

  const clearError = () => {
    setError(null);
  };

  return {
    clearError,
    download,
    error,
    isDownloading,
  };
};

export { useDownloadValues, getValues };
