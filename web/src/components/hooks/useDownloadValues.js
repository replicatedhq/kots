import { useState, useRef } from "react";
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
        "Authorization": _token,
        "Content-Type": "application/json",
      }
    });

    const data = await response.json();
    console.log(data);
    return { data };
  } catch (error) {
    return { error };
  }
}

const useDownloadValues = ({
  _createObjectURL = URL.createObjectURL,
  _getValues = getValues,
  _revokeObjectURL = URL.revokeObjectURL,
  appSlug,
  fileName,
} = {}) => {
  const ref = useRef(null);
  const [isDownloading, setIsDownloading] = useState(false);
  const [error, setError] = useState(null);
  const [url, setUrl] = useState(null);
  const [name, setName] = useState(null);
  console.log(name);
  console.log(url);

  const download = async () => {
    try {
      setIsDownloading(true);
      setError(null);
      const { data } = await _getValues({
        appSlug,
      });
      const url = _createObjectURL(new Blob([data]));
      setUrl(url)
      setName(fileName);
      ref.current?.click();

      setIsDownloading(false);
      _revokeObjectURL(url)

    } catch (error) {
      setIsDownloading(false);
      setError(error);
    }
  };

  return {
    download,
    error,
    isDownloading,
    name,
    ref,
    url,
  };
}

export { useDownloadValues };

