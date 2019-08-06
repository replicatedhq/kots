import pg from "pg";
import bcrypt from "bcrypt";
import randomstring from "randomstring";
import { User } from "./";
import { logger } from "../server/logger";

export class UserStore {
  constructor(
    private readonly pool: pg.Pool,
  ) {}

  public async migrateUsers(): Promise<void> {
    const pg = await this.pool.connect();

    console.log("migating users");
    try {
      await pg.query("begin");

      const q = `select id, created_at, github_id from ship_user`;
      const v = [];

      const result = await pg.query(q, v);
      for (const row of result.rows) {
        if (row.github_id) {
          console.log(`migrating user ${row}`);
          const qq = `update github_user set user_id = $1 where github_id = $2`;
          const vv = [
            row.id,
            row.github_id,
          ];
          await pg.query(qq, vv);
        }
      }

      console.log("commiting");
      await pg.query("commit");
    } catch (err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

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

  public async tryGetPasswordUser(email: string): Promise<User | void> {
    const q = `select user_id from ship_user_local where email = $1`;
    const v = [
      email,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount === 0) {
      return;
    }

    return this.getUser(result.rows[0].user_id);
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
    const user: User = new User();

    let q = `select id, created_at from ship_user where id = $1`;
    let v = [id];
    let result = await this.pool.query(q, v);

    user.id = result.rows[0].id;
    user.createdAt = result.rows[0].created_at;

    // GitHub
    q = `select username, github_id, avatar_url, email from github_user where user_id = $1`;
    v = [id];
    result = await this.pool.query(q, v);
    if (result.rowCount > 0) {
      user.githubUser = {
        login: result.rows[0].username,
        githubId: result.rows[0].github_id,
        avatarUrl: result.rows[0].avatar_url,
        email: result.rows[0].email,
      };
    }

    // Ship
    q = `select email, first_name, last_name, password_bcrypt from ship_user_local where user_id = $1`;
    v = [id];
    result = await this.pool.query(q, v);
    if (result.rowCount > 0) {
      user.shipUser = {
        firstName: result.rows[0].first_name,
        lastName: result.rows[0].last_name,
        email: result.rows[0].email,
        passwordCrypt: result.rows[0].password_bcrypt,
      };
    }

    return user;
  }

  public async createGitHubUser(githubId: number, githubLogin: string, githubAvatar: string, email: string): Promise<User> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      let v: any[] = [];

      let q = `insert into ship_user (id, created_at, last_login) values ($1, $2, $3)`;
      v = [
          id,
          new Date(),
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

      let q = `insert into ship_user (id, created_at, last_login) values ($1, $2, $3)`;
      let v = [
          id,
          new Date(),
          new Date(),
      ];
      await pg.query(q, v);

      q = `insert into ship_user_local (user_id, password_bcrypt, first_name, last_name, email) values ($1, $2, $3, $4, $5)`;
      v = [
        id,
        await bcrypt.hash(password, 10),
        firstName,
        lastName,
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

  async trackScmLead(preference: string, email: string, provider: string): Promise<string> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const currentTime = new Date(Date.now()).toUTCString();

    const q = `insert into track_scm_leads (id, deployment_type, email_address, scm_provider, created_at)
      values ($1, $2, $3, $4, $5)`;
    const v = [
      id,
      preference,
      email,
      provider,
      currentTime
    ];

    await this.pool.query(q, v);

    return id;
  }

  async updateLastLogin(id: string): Promise<boolean> {
    const currentTime = new Date();

    const q = "UPDATE ship_user SET last_login = $1 where id = $2";

    const v = [
      currentTime,
      id
    ];

    await this.pool.query(q, v);

    return true;
  }

  public async checkSecuredStatus(): Promise<Boolean> {
    const q = "select count(1) as count from kotsadm_params";

    const result = await this.pool.query(q);

    if (result.rows[0].count === "0") {
      return false;
    }

    return true;
  }

  public async createAdminConsolePassword(password: string): Promise<string> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();
    const encryptedPassword = await bcrypt.hash(password, 10);
    try {
      await pg.query("begin");

      let q = `insert into kotsadm_params (key, password_bcrypt) values ($1, $2)`;
      let v = [ "secure-password", encryptedPassword ];
      await pg.query(q, v);

      q = `insert into ship_user (id, created_at, last_login) values ($1, $2, $3)`;
      v = [
        id,
        new Date(),
        new Date(),
      ];
      await pg.query(q, v);

      q = `insert into ship_user_local (user_id, password_bcrypt, first_name, last_name, email) values ($1, $2, $3, $4, $5)`;
      v = [
        id,
        encryptedPassword,
        "Default",
        "User",
        "default-user@none.com",
      ];
      await pg.query(q, v);

      await pg.query("commit");

      return id;
    } catch(err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

}
