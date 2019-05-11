// @ts-ignore
import * as jaeger from "jaeger-client";
import { get, isFunction } from "lodash";
import * as opentracing from "opentracing";
import { logger } from "./logger";

const initTracer = jaeger.initTracer;

let tracerInstance: any;

export function bootstrap(): any {
  const jaegerLogger = logger.child({ lib: "jaeger" });
  const name = process.env.TRACER_NAME || "ship-cluster-api";
  const config = {
    serviceName: name,
    sampler: {
      type: "const",
      param: 1,
    },
    reporter: {
      agentHost: "jaeger-agent",
      agentPort: 6832,
      logSpans: true,
    },
  };
  const options = {
    tags: {
      name: "unknown",
    },
    // temporary -- quiet these down, they are really noisy in production. Once jaeger is hooked up in EKS,
    // we can consider turning the `error` events back on
    // implements https://github.com/jaegertracing/jaeger-client-node/blob/master/src/_flow/logger.js
    logger: {
      info: msg => {
        jaegerLogger.debug(msg);
      },
      error: msg => {
        jaegerLogger.debug(msg);
      },
    },
  };
  tracerInstance = initTracer(config, options);
}

export function tracer(): opentracing.Tracer {
  if (!tracerInstance) {
    bootstrap();
  }
  return tracerInstance;
}

// this is janky but it seems like kind of what you need to do
// decorators inherently require global state since they're evaluated
// at import time.
//
// right now this is only called in tests, so we can swap in a mock tracer
export function setTracer(t: opentracing.Tracer) {
  tracerInstance = t;
}

export type TraceFunc<T> = (spanContext: opentracing.SpanContext) => Promise<T>;

export class FuncTracer {
  constructor(private readonly client: opentracing.Tracer) {}

  async trace<T>({ tags, name, parentCtx }: TraceOpts, f: TraceFunc<T>): Promise<T> {
    const theName = name || "unnamed";
    const span = parentCtx ? this.client.startSpan(theName, { childOf: parentCtx }) : this.client.startSpan(theName);

    for (const tag of Object.keys(tags || {})) {
      span.setTag(tag, tags![tag]);
    }

    try {
      return await f(span.context());
    } catch (err) {
      span.setTag(opentracing.Tags.ERROR, true);
      span.log({ event: "error", "error.object": err, message: err.message, stack: err.stack });
      throw err;
    } finally {
      span.finish();
    }
  }
}

export interface TraceDecoratorOpts {
  tags?: {
    [key: string]: string;
  };
  name?: string;
  paramTags?: {
    [key: string]: string | number;
  };
}

export interface TraceOpts {
  tags?: {
    [key: string]: string;
  };
  name: string;
  parentCtx?: opentracing.SpanContext;
}

export function traced(opts?: TraceDecoratorOpts) {
  const tags = (opts && opts.tags) || {};
  const paramTags = (opts && opts.paramTags) || {};

  return (target: any, key: string, inputDescriptor: PropertyDescriptor | undefined) => {
    let descriptor = inputDescriptor;
    if (descriptor === undefined) {
      descriptor = Object.getOwnPropertyDescriptor(target, key);
    }
    const originalMethod = descriptor!.value;
    const klass = target.constructor.name;
    tags.class = klass;
    tags.method = key;

    // this needs to be a non-arrow function or we'll get the wrong `this`
    function overrideMethod() {
      // tslint:disable-next-line
      const self = this;

      const name = (opts && opts.name) || `${klass}.${key}`;
      const args = arguments;

      const maybeContext: jaeger.SpanContext = args[0];
      const contextClass = maybeContext && maybeContext.constructor && maybeContext.constructor.name;
      if (!isFunction(maybeContext.withBaggageItem)) {
        // this is a weird way to check, but instanceof breaks tests because test mocks are ES5Proxy instances
        logger.warn({
          klass,
          key,
          msg: "@traced() on method without jaeger.SpanContext as first arg",
          received: contextClass,
        });
        return originalMethod.apply(self, args);
      }

      const tagsFromArguments: string[] = Object.keys(paramTags);
      tagsFromArguments.forEach((tagKey: string) => {
        const pathToValue: string | number = paramTags[tagKey];
        const tagValue: any = get(args, pathToValue);
        if (tagValue && isFunction(tagValue.toString)) {
          tags[tagKey] = tagValue.toString();
        }
      });

      return new FuncTracer(tracer()).trace({ name, parentCtx: maybeContext, tags }, async (ctx: opentracing.SpanContext) => {
        // subsitute in the child context
        args[0] = ctx;
        return originalMethod.apply(self, args);
      });
    }

    descriptor!.value = overrideMethod;

    return descriptor;
  };
}