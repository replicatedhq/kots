import randomstring from "randomstring";
import pg from "pg";
import { logger } from "../server/logger";

export class GithubInstallationsStore {
  constructor(private readonly pool: pg.Pool) {}

  async createNewGithubInstall(installationId: number, accountLogin: string, accountUrl: string, initialLogin: string, initialUrl: string): Promise<void> {
    const q = `insert into github_install
      (id, installation_id, account_login, account_html_url, sender_login, sender_html_url, created_at)
      values
      ($1, $2, $3, $4, $5, $6, $7)`;
    const id = randomstring.generate({ capitalization: "lowercase" });
    const v = [
      id, 
      installationId, 
      accountLogin, 
      accountUrl, 
      initialLogin, 
      initialUrl, 
      new Date()
    ];
    await this.pool.query(q, v);
  }

  async deleteGithubInstall(installationId: number): Promise<void> {
    const q = `delete from github_install where installation_id = $1`;
    const v = [ installationId];
    await this.pool.query(q, v);
  }
}