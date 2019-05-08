import { Service } from "ts-express-decorators";
import { CreateUpdateSessionMutationArgs, UpdateSession } from "../generated/types";
import { Mutation } from "../schema/decorators";
import { Context } from "../context";
import { tracer } from "../server/tracing";
import { UpdateStore } from "./store";

@Service()
export class Update {
  constructor(private readonly updateStore: UpdateStore) {}

  @Mutation("ship-cloud")
  async createUpdateSession(root: any, { watchId }: CreateUpdateSessionMutationArgs, context: Context): Promise<UpdateSession> {
    const span = tracer().startSpan("mutation.createUpdateSession");

    const updateSession = await this.updateStore.createUpdateSession(span.context(), context.session.userId, watchId);
    const deployedUpdateSession = await this.updateStore.deployUpdateSession(span.context(), updateSession.id!);

    span.finish();

    return deployedUpdateSession;
  }
}
