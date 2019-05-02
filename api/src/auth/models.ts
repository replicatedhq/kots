export interface UserModel {
  id: string;
  github_username: string | null;
  bitbucket_username: string | null;
  gitlab_username: string | null;
  email: string | null;
}

export interface ScmLeadModel {
  id: string;
  deployment_type: string | null;
  email_address: string | null;
  scm_provider: string | null;
  created_at: string | null;
  followed_up: boolean | null;
}

export interface GithubNonceModel {
  nonce: string;
  expire_at?: string;
}
