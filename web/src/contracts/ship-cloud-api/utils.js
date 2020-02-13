import * as jwt from "jsonwebtoken";
import { ShipClientGQL } from "../../ShipClientGQL";
import fetch from "node-fetch";

const SESSION_KEY = "testsession";
const MOCK_SERVER_PORT = 3333;

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

export function getShipClient(sessionId) {
  return ShipClientGQL(
    `:${MOCK_SERVER_PORT}/graphql`,
    `:${MOCK_SERVER_PORT}/api`,
    () => {
      if (!sessionId) {
        return "";
      }
      return createSessionToken(sessionId)
    }, fetch);
}