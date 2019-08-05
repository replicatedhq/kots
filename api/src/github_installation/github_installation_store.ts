import randomstring from "randomstring";
import pg from "pg";

export class GithubInstallationsStore {
  constructor(private readonly pool: pg.Pool) {}

  async createNewGithubInstall(installationId: number, accountLogin: string, accountType: string, numberOfOrgMembers: number, accountUrl: string, initialLogin: string): Promise<void> {
    const q = `insert into github_install
      (id, installation_id, account_login, account_type, organization_members_total, account_html_url, sender_login, created_at)
      values
      ($1, $2, $3, $4, $5, $6, $7, $8)`;
    const id = randomstring.generate({ capitalization: "lowercase" });
    const v = [
      id, 
      installationId, 
      accountLogin, 
      accountType,
      numberOfOrgMembers,
      accountUrl, 
      initialLogin, 
      new Date()
    ];
    await this.pool.query(q, v);
  }

  async deleteGithubInstall(installationId: number): Promise<void> {
    const q = `update github_install set is_deleted = true where installation_id = $1`;
    const v = [ installationId ];
    await this.pool.query(q, v);
  }
}