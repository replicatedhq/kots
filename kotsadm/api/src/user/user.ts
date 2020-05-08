import bcrypt from "bcrypt";
import { ReplicatedError } from "../server/errors";

export class User {
  public id: string
  public createdAt: string;
  public lastLogin: string;

  public githubUser?: GitHubUser;
  public shipUser?: ShipUser;
}

export interface GitHubUser {
  login: string;
  githubId: number;
  avatarUrl: string;
  email: string;
}

export interface ShipUser {
  firstName: string;
  lastName: string;
  email: string;
}

export interface AccessToken {
  access_token: string;
}

export interface AdminSignupInfo {
  token: string;
  userId: string;
}
