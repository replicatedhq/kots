import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { VeleroClient } from "./veleroClient";

export function SnapshotMutations(stores: Stores) {
  return {
    async deleteSnapshot(root: any, args: any, context: Context): Promise<void> {
      context.requireSingleTenantSession();
      const velero = new VeleroClient("velero"); // TODO namespace
      await velero.deleteSnapshot(args.snapshotName);
    },
  };
}
