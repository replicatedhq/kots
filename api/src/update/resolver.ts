import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { authorized } from "../auth/decorators";
import { CreateUpdateSessionMutationArgs, UpdateSession } from "../generated/types";
import { Mutation } from "../schema/decorators";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { UpdateStore } from "./store";

@Service()
export class Update {
  constructor(private readonly updateStore: UpdateStore) {}

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async createUpdateSession(root: any, { watchId }: CreateUpdateSessionMutationArgs, context: Context): Promise<UpdateSession> {
    const span = tracer().startSpan("mutation.createUpdateSession");

    const updateSession = await this.updateStore.createUpdateSession(span.context(), context.userId, watchId);
    const deployedUpdateSession = await this.updateStore.deployUpdateSession(span.context(), updateSession.id!);

    span.finish();

    return deployedUpdateSession;
  }
}
