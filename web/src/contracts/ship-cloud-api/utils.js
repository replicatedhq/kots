import * as jwt from "jsonwebtoken";

const SESSION_KEY = "testsession";

export function createSessionToken(sessionId) {
  return jwt.sign(
    {
      iat: new Date(Date.UTC(2019, 0, 1, 1, 0, 0)).getTime(),
      token: "not-checked",
      sessionId,
    },
    SESSION_KEY
  );
}
