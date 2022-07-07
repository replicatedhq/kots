import _fetch from "isomorphic-fetch";
import { useState, useRef } from "react";
import { Utilities } from "../../utilities/utilities";

const putConfig = async ({
  _token = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
  body
}) => {

  try {
    const response = await fetch(`${apiEndpoint}/app/${appSlug}/config`, {
      method: "PUT",
      headers: {
        "Authorization": _token,
        "Content-Type": "application/json",
      },
      body
    });

    const data = await response.json();
    return { data };
  } catch (error) {
    return { error };
  }
}

const useSaveConfig = ({
  _putConfig = putConfig,
  appSlug,
} = {}) => {
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState(null);

  const saveConfig = async ({
    body
  }) => {

    try {
      setIsSaving(true);
      setError(null);
      const { data } = await _putConfig({
        appSlug,
        body,
      });

      setIsSaving(false);

    } catch (error) {
      setIsSaving(false);
      setError(error);
    }
  };
  return { saveConfig, isSaving, error };
}

export { useSaveConfig };