import * as jaeger from "jaeger-client";
import * as jwt from "jsonwebtoken";
import { expect } from "chai";
import { afterEach, beforeEach, describe, it } from "mocha";
import { IMock, It, Mock, Times } from "typemoq";
import { Params } from "../server/params";
import { Context } from "../server/server";
import { Session } from "./index";
import { SessionStore } from "./store";
import { SessionModel } from "./models";

describe("Session", () => {
  let session: Session;
  let params: Params;
  let sessionStore: IMock<SessionStore>;
  let tracer: IMock<jaeger.Tracer>;
  let span: IMock<jaeger.SpanContext>;

  beforeEach(() => {
    sessionStore = Mock.ofType(SessionStore);
    tracer = Mock.ofType(jaeger.Tracer);

    span = Mock.ofType<any>();

    tracer
      .setup(x => x.startSpan("session.decode"))
      .returns(() => span.object)
      .verifiable(Times.once());

    span.setup(x => x.finish()).verifiable(Times.once());
  });

  afterEach(() => {
    sessionStore.verifyAll();
    tracer.verifyAll();
    span.verifyAll();
  });

  it("skips an empty token", async () => {
    params = new Params({} as any);
    session = new Session(params, sessionStore.object, tracer.object);

    const ctx: Context = await session.decode("");
    expect(ctx.auth).to.equal("");
    expect(ctx.sessionId).to.equal("");
    expect(ctx.userId).to.equal("");
  });

  it('skips a token equal to "null"', async () => {
    params = new Params({} as any);
    session = new Session(params, sessionStore.object, tracer.object);

    const ctx: Context = await session.decode("null");
    expect(ctx.auth).to.equal("");
    expect(ctx.sessionId).to.equal("");
    expect(ctx.userId).to.equal("");
  });

  it("decodes a valid token", async () => {
    const sessionKey = "not-so-secure";
    const sessionId = "sessionId";
    const token = "token";
    const sessionModel: SessionModel = {
      id: sessionId,
      metadata: `{"foo":"bar"}`,
      user_id: "userId",
      expiry: new Date(),
    };

    params = new Params({ sessionKey } as any);

    sessionStore
      .setup(x => x.getGithubSession(It.isAny(), sessionId))
      .returns(() => Promise.resolve(sessionModel))
      .verifiable(Times.once());

    const sess = new Session(params, sessionStore.object, tracer.object);

    const signedJWT = await new Promise<string>((resolve, reject) => {
      jwt.sign({ token, sessionId }, sessionKey, (err, encoded) => {
        if (err) {
          reject(err);
          return;
        }
        resolve(encoded);
      });
    });

    const ctx: Context = await sess.decode(signedJWT);
    expect(ctx.auth).to.equal("token");
    expect(ctx.sessionId).to.equal("sessionId");
    expect(ctx.userId).to.equal("userId");
    expect(ctx.metadata).to.deep.equal({ foo: "bar" });
  });
});
