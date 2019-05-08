import { PostgresWrapper } from "../util/persistence/db";
import * as randomstring from "randomstring";
import { User } from "./user";
import { Service } from "ts-express-decorators";

@Service()
export class UserStore {
  constructor(readonly wrapper: PostgresWrapper) {}

  public async listAllUsers(): Promise<User[]> {
    const q = `select id from ship_user`;
    const v = [];

    const result = await this.wrapper.query(q, v);
    const users: User[] = [];
    for (const row of result.rows) {
      const user = await this.getUser(row.id);
      users.push(user);
    }

    return users;
  }

  public async getUser(id: string): Promise<User> {
    const q = `select id, created_at from ship_user where id = $1`;
    const v = [id];

    const result = await this.wrapper.query(q, v);
    const user: User = new User();
    user.id = result.rows[0].id;
    user.createdAt = result.rows[0].created_at;
    return user;
  }

  public async createGitHubUser(githubId: number, githubLogin: string, githubAvatar: string, email: string): Promise<User> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    await this.wrapper.query("begin");

    try {
      let v: any[] = [];

      let q = `insert into ship_user (id, created_at) values ($1, $2)`;
      v = [
          id,
          new Date(),
      ];
      await this.wrapper.query(q, v);

      q = `insert into github_user (user_id, username, github_id, avatar_url, email) values ($1, $2, $3, $4, $5)`;
      v = [
        id,
        githubLogin,
        githubId,
        githubAvatar,
        email,
      ];
      await this.wrapper.query(q, v);

      await this.wrapper.query("commit");
    } catch {
        await this.wrapper.query("rollback");
    }

    return this.getUser(id);
  }

  public async createPasswordUser(email: string, password: string, firstName: string, lastName: string): Promise<User> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    await this.wrapper.query("begin");

    try {
      let q = `insert into ship_user (id, created_at) values ($1, $2)`;
      let v = [
          id,
          new Date(),
      ];
      await this.wrapper.query(q, v);

      await this.wrapper.query("commit");
    } catch {
        await this.wrapper.query("rollback");
    }

    return this.getUser(id);    
  }
}