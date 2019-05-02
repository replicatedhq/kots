import { expect } from "chai";
import * as jaeger from "jaeger-client";
import { afterEach, beforeEach, describe, it } from "mocha";
import * as opentracing from "opentracing";
import { IMock, It, Mock, MockBehavior, Times } from "typemoq";
import { FuncTracer, setTracer, traced } from "./tracing";

describe("tracing", () => {
  let tracer: IMock<opentracing.Tracer>;
  let span: IMock<opentracing.Span>;
  let parentCtx: IMock<opentracing.SpanContext>;
  let childCtx: IMock<opentracing.SpanContext>;

  beforeEach(() => {
    tracer = Mock.ofType(opentracing.Tracer, MockBehavior.Strict);
    parentCtx = Mock.ofType(jaeger.SpanContext);
    span = Mock.ofType(opentracing.Span, MockBehavior.Strict);
    childCtx = Mock.ofType(opentracing.SpanContext, MockBehavior.Strict);
  });

  afterEach(() => {
    tracer.verifyAll();
    parentCtx.verifyAll();
    span.verifyAll();
    childCtx.verifyAll();
  });

  describe("FuncTracer", () => {
    it("trace(opts, traceFunc)", async () => {
      tracer
        .setup(x => x.startSpan("lol", { childOf: parentCtx.object }))
        .returns(() => span.object)
        .verifiable(Times.once());

      span.setup(x => x.setTag("kfbr", "392")).verifiable(Times.once());
      span
        .setup(x => x.context())
        .returns(() => childCtx.object)
        .verifiable(Times.once());

      span.setup(x => x.finish()).verifiable(Times.once());

      const ret = await new FuncTracer(tracer.object).trace<string>(
        {
          parentCtx: parentCtx.object,
          name: "lol",
          tags: { kfbr: "392" },
        },
        async spanContext => {
          expect(spanContext).to.equal(childCtx.object);
          return "foo";
        },
      );
      expect(ret).to.equal("foo");
    });

    it("trace(opts, traceFunc) with error", async () => {
      const err = new Error("lol");

      tracer
        .setup(x => x.startSpan("lol", { childOf: parentCtx.object }))
        .returns(() => span.object)
        .verifiable(Times.once());

      span.setup(x => x.setTag("kfbr", "392")).verifiable(Times.once());
      span
        .setup(x => x.context())
        .returns(() => childCtx.object)
        .verifiable(Times.once());
      span.setup(x => x.finish()).verifiable(Times.once());
      span.setup(s => s.setTag(opentracing.Tags.ERROR, true)).verifiable(Times.once());
      span
        .setup(s =>
          s.log({
            event: "error",
            "error.object": err,
            message: err.message,
            stack: err.stack,
          }),
        )
        .verifiable(Times.once());

      try {
        await new FuncTracer(tracer.object).trace<string>(
          {
            parentCtx: parentCtx.object,
            name: "lol",
            tags: { kfbr: "392" },
          },
          spanContext => {
            expect(spanContext).to.equal(childCtx.object);
            throw err;
          },
        );
        throw new Error("expected error not thrown");
      } catch (e) {
        expect(e).to.equal(err);
      }

      tracer.verifyAll();
      parentCtx.verifyAll();
      span.verifyAll();
      childCtx.verifyAll();
    });
  });

  describe("@traced", () => {
    it("decorates", async () => {
      class Traceable {
        @traced({ tags: { kfbr: "392" } })
        async doExpensiveWork(ctx: opentracing.SpanContext): Promise<String> {
          return new Promise<string>((resolve, reject) => {
            try {
              expect(ctx).to.equal(childCtx.object);
            } catch (err) {
              reject(err);
            }
            setTimeout(() => resolve("we did it"), 100);
          });
        }
      }

      tracer
        .setup(x => x.startSpan("Traceable.doExpensiveWork", { childOf: parentCtx.object }))
        .returns(() => span.object)
        .verifiable(Times.once());

      span.setup(x => x.setTag("kfbr", "392")).verifiable(Times.once());
      span.setup(x => x.setTag("class", "Traceable")).verifiable(Times.once());
      span.setup(x => x.setTag("method", "doExpensiveWork")).verifiable(Times.once());
      span
        .setup(x => x.context())
        .returns(() => childCtx.object)
        .verifiable(Times.once());
      span.setup(x => x.finish()).verifiable(Times.once());

      setTracer(tracer.object);

      const result = await new Traceable().doExpensiveWork(parentCtx.object);
      expect(result).to.equal("we did it");
    });

    it("decorates with an error", async () => {
      const error = new Error("whoops!");

      class Traceable {
        @traced({ tags: { kfbr: "392" } })
        async doExpensiveWork(ctx: opentracing.SpanContext) {
          return new Promise((resolve, reject) => {
            reject(error);
          });
        }
      }

      tracer
        .setup(x => x.startSpan("Traceable.doExpensiveWork", { childOf: parentCtx.object }))
        .returns(() => span.object)
        .verifiable(Times.once());

      span.setup(x => x.setTag("kfbr", "392")).verifiable(Times.once());
      span.setup(x => x.setTag("class", "Traceable")).verifiable(Times.once());
      span.setup(x => x.setTag("method", "doExpensiveWork")).verifiable(Times.once());
      span
        .setup(x => x.context())
        .returns(() => childCtx.object)
        .verifiable(Times.once());
      span.setup(x => x.finish()).verifiable(Times.once());
      span.setup(s => s.setTag(opentracing.Tags.ERROR, true)).verifiable(Times.once());
      span
        .setup(s =>
          s.log({
            event: "error",
            "error.object": error,
            message: error.message,
            stack: error.stack,
          }),
        )
        .verifiable(Times.once());

      setTracer(tracer.object);

      try {
        await new Traceable().doExpensiveWork(parentCtx.object);
        throw new Error("expected error was not thrown");
      } catch (err) {
        expect(err).to.equal(error);
      }
    });

    it("falls back to original method if first arg is not a spanContext", async () => {
      class Untraceable {
        @traced({ tags: { kfbr: "392" } })
        async doExpensiveWork(userId: string): Promise<string> {
          return userId;
        }
      }
      setTracer(tracer.object);
      const result = await new Untraceable().doExpensiveWork("some id");
      expect(result).to.equal("some id");
    });
  });

  it("decorates with tags from parameters", async () => {
    class Traceable {
      @traced({ tags: { kfbr: "392" }, paramTags: { userId: "1", watchId: "2" } })
      async doExpensiveWork(ctx: opentracing.SpanContext, user: string, watch: string): Promise<String> {
        return new Promise<string>((resolve, reject) => {
          try {
            expect(ctx).to.equal(childCtx.object);
          } catch (err) {
            reject(err);
          }
          setTimeout(() => resolve("we did it"), 100);
        });
      }
    }

    tracer
      .setup(x => x.startSpan("Traceable.doExpensiveWork", { childOf: parentCtx.object }))
      .returns(() => span.object)
      .verifiable(Times.once());

    span.setup(x => x.setTag("kfbr", "392")).verifiable(Times.once());
    span.setup(x => x.setTag("class", "Traceable")).verifiable(Times.once());
    span.setup(x => x.setTag("method", "doExpensiveWork")).verifiable(Times.once());
    span.setup(x => x.setTag("userId", "some-user-id")).verifiable(Times.once());
    span.setup(x => x.setTag("watchId", "some-watch-id")).verifiable(Times.once());
    span
      .setup(x => x.context())
      .returns(() => childCtx.object)
      .verifiable(Times.once());
    span.setup(x => x.finish()).verifiable(Times.once());

    setTracer(tracer.object);

    const result = await new Traceable().doExpensiveWork(parentCtx.object, "some-user-id", "some-watch-id");
    expect(result).to.equal("we did it");
  });
});
