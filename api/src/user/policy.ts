import { isAfter } from "date-fns";
import { ReplicatedError } from "../server/errors";
import { Context } from "../context";

export function isPolicyValid(context: Context): ReplicatedError | null {
  if (context.getGitHubToken().length === 0) {
    return new ReplicatedError("Unauthorized", "401");
  }

  const currentTime = new Date(Date.now()).toUTCString();
  if (isAfter(currentTime, context.session.expiresAt)) {
    return new ReplicatedError("Expired session", "401");
  }

  return null;
}
