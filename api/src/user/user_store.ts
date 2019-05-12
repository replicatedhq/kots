import * as pg from "pg";
import * as bcrypt from "bcrypt";
import * as randomstring from "randomstring";
import { User } from "./user";

export class UserStore {
  constructor(
    private readonly pool: pg.Pool,
  ) {}

  public async listAllUsers(): Promise<User[]> {
    const q = `select id from ship_user`;
    const v = [];

    const result = await this.pool.query(q, v);
    const users: User[] = [];
    for (const row of result.rows) {
      const user = await this.getUser(row.id);
      users.push(user);
    }

    return users;
  }

  public async tryGetGitHubUser(githubId: number): Promise<User | void> {
    const q = `select user_id from github_user where github_id = $1`;
    const v = [
      githubId,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount === 0) {
      return;
    }

    return this.getUser(result.rows[0].user_id);
  }

  public async getUser(id: string): Promise<User> {
    const q = `select id, created_at from ship_user where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);
    const user: User = new User();
    user.id = result.rows[0].id;
    user.createdAt = result.rows[0].created_at;
    return user;
  }

  public async createGitHubUser(githubId: number, githubLogin: string, githubAvatar: string, email: string): Promise<User> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      let v: any[] = [];

      let q = `insert into ship_user (id, created_at) values ($1, $2)`;
      v = [
          id,
          new Date(),
      ];
      await pg.query(q, v);

      q = `insert into github_user (user_id, username, github_id, avatar_url, email) values ($1, $2, $3, $4, $5)`;
      v = [
        id,
        githubLogin,
        githubId,
        githubAvatar,
        email,
      ];
      await pg.query(q, v);

      await pg.query("commit");

      return this.getUser(id);
    } catch(err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

  public async createPasswordUser(email: string, password: string, firstName: string, lastName: string): Promise<User> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      let q = `insert into ship_user (id, created_at) values ($1, $2)`;
      let v = [
          id,
          new Date(),
      ];
      await pg.query(q, v);

      q = `insert into ship_user_local (user_id, password_bcrypt, first_name, last_name) values ($1, $2, $3, $4)`;
      v = [
        id,
        await bcrypt.hash(password, 10),
        firstName,
        lastName,
      ];
      await pg.query(q, v);

      await pg.query("commit");

      return this.getUser(id);
    } catch(err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }
}
