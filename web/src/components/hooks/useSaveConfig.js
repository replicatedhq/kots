import _fetch from "isomorphic-fetch";
import { useState, useEffect } from "react";
import { Utilities } from "../../utilities/utilities";

export const putConfig = async ({
  _token = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
}) => {


  const { fromLicenseFlow, history, match } = this.props;
  const sequence = this.getSequence();
  const createNewVersion = !fromLicenseFlow && match.params.sequence == undefined;

  fetch(`${apiEndpoint}/app/${appSlug}/config`, {
    method: "PUT",
    headers: {
      "Authorization": _token,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      configGroups: this.state.configGroups,
      sequence,
      createNewVersion,
    })
  })
    .then(res => res.json())
    .then(async (result) =>

  return Promise.resolve();
}

export const useSaveConfig = ({
  _createObjectURL = window.URL.createObjectURL,
  _putConfig = putConfig,
  _revoteObjectUrl = window.URL.revokeObjectURL,
  appSlug,
  fileName,
  onError,
  putStart = () => {},
  putFinished = () => {},
}) => {
  const ref = useRef(null);
  const [url, setFileUrl] = useState();
  const [name, setFileName] = useState();

  const saveConfig = async ({
    body
  }) => {
    try {
      putStart()
      const { data } = await _putConfig({
        appSlug,
        body,
      });

      // create download object
      const url = _createObjectURL(new Blob([data]));
      setFileUrl(url);
      setFileName(getFileName());

      // trigger donwload file
      ref.current?.click();

      postDownloading();

      _revoteObjectUrl(url);
    } catch (error) {
      onError();
    }
  };
  return { saveConfg, ref, url, name };
}