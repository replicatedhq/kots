import { useMutation } from "react-query";
import { Utilities } from "../../utilities/utilities";

const putConfig = async ({
  _fetch = fetch,
  _token = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  appSlug,
  body,
}) => {
  try {
    const response = await _fetch(`${apiEndpoint}/app/${appSlug}/config`, {
      method: "PUT",
      headers: {
        Authorization: _token,
        "Content-Type": "application/json",
      },
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
