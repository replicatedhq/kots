import { logger } from "../server/logger";
import { Context } from "../server/server";
import { isPolicyValid } from "./policy";

export function authorized() {
  return (target: any, key: string, inDescriptor: PropertyDescriptor | undefined) => {
    let descriptor = inDescriptor;
    if (descriptor === undefined) {
      descriptor = Object.getOwnPropertyDescriptor(target, key);
    }
    if (descriptor === undefined) {
      return descriptor;
    }

    const originalMethod = descriptor.value;

    function enforcePolicy(...args: any[]) {
      const context: Context = args[2];

      const error = isPolicyValid(context);
      if (error) {
        throw error;
      }

      logger.debug("args", args);

      // tslint:disable-next-line:no-invalid-this
      return originalMethod.apply(this, args);
    }

    descriptor.value = enforcePolicy;

    return descriptor;
  };
}
