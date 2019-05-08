import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { authorized } from "../user/decorators";
import { Mutation, Query } from "../schema/decorators";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { UnforkStore } from "./unfork_store";
import { CreateUnforkSessionMutationArgs, UnforkSession, UpdateSession } from "../generated/types";
import { UpdateStore } from "../update/store";

interface Error {
  result: string
}

@Service()
export class Unfork {
  constructor(
    private readonly unforkStore: UnforkStore,
    private readonly updateStore: UpdateStore,
  ) {}

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async createUnforkSession(root: any, args: CreateUnforkSessionMutationArgs, context: Context): Promise<Error|UpdateSession> {
    const span = tracer().startSpan("mutation.createUnforkSession");

    const { upstreamUri, forkUri } = args;

    const unforkSession = await this.unforkStore.createUnforkSession(span.context(), context.userId, upstreamUri, forkUri);
    const deployedUnforkSession = await this.unforkStore.deployUnforkSession(span.context(), unforkSession.id!);

    // Until we have unfork headed mode, we just create an update haded job to allow for UI
    const now = new Date();
    const abortAfter = new Date(now.getTime() + (1000 * 60));
    while (new Date() < abortAfter) {
      const maybeUpdatedSession = await this.unforkStore.getSession(span.context(), deployedUnforkSession.id!);
      if (maybeUpdatedSession.result === "failed") {
        span.finish();

        return {
          result: "error unforking application",
        };

      } else if (maybeUpdatedSession.result === "completed") {
        const updateSession = await this.updateStore.createUpdateSession(span.context(), context.userId, maybeUpdatedSession.id!);
        const deployedUpdateSession = await this.updateStore.deployUpdateSession(span.context(), updateSession.id!);

        span.finish();

        return deployedUpdateSession;
      }

      await sleep(100);
    };

    span.finish();

    return {
      result: "timeout unforking application",
    };
  }
}

// hack
function sleep(ms = 0) {
  return new Promise(r => setTimeout(r, ms));
}
