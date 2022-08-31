import { useState, useEffect } from "react";
import { Utilities } from "../../utilities/utilities";

const getValues = async ({
  _fetch = fetch,
  _token = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
}) => {
  try {
    const response = await _fetch(`${apiEndpoint}/app/${appSlug}/values`, {
      method: "GET",
      headers: {
        Authorization: _token,
        "Content-Type": "application/blob",
      },
    });
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
      const { data, error: _error } = await _getValues({
        appSlug,
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

function UseDownloadValues({ appSlug, fileName, children }) {
  const query = useDownloadValues({ appSlug, fileName });

  return children(query);
}

export { useDownloadValues, UseDownloadValues, getValues };
