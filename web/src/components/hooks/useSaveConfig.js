import { useMutation } from "@tanstack/react-query";

const putConfig = async ({
  _fetch = fetch,
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
  body,
}) => {
  try {
    const response = await _fetch(`${apiEndpoint}/app/${appSlug}/config`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body,
    });

    if (!response.ok) {
      throw new Error("Error saving config");
    }

    const data = await response.json();
    return { data };
  } catch (error) {
    throw Error(error);
  }
};

const useSaveConfig = ({ _putConfig = putConfig, appSlug } = {}) =>
  useMutation(({ body }) => _putConfig({ appSlug, body }));

export { useSaveConfig, putConfig };
