import { Utilities } from "@src/utilities/utilities";

export const getAppsList = async () => {
  try {
    const res = await fetch(`${process.env.API_ENDPOINT}/apps`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "GET",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      console.log("failed to list apps, unexpected status code", res.status);
      return;
    }
    const response = await res.json();
    const apps = response.apps;
    // setState({
    //   appsList: apps,
    // });
    return apps;
  } catch (err) {
    throw err;
  }
};
