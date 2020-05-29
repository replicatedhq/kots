import jwt from "jsonwebtoken";
import { Params } from "../server/params";

interface Claims {
  session_id: string;
  user_id: string;
}

export class Session {
  public sessionId: string;
  public userId: string;
  public expiresAt: Date;
  public metadata: string;
  public type: string;

  public async getToken(): Promise<string> {
    const claims: Claims = {
      session_id: this.sessionId,
      user_id: this.userId,
    };

    const params = await Params.getParams();
    const token = jwt.sign(claims, params.sessionKey);
    return token;
  }
}
