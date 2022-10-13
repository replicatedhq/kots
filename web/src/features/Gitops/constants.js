import styled from "styled-components";

export const SERVICES = [
  {
    value: "github",
    label: "GitHub",
  },
  {
    value: "github_enterprise",
    label: "GitHub Enterprise",
  },
  {
    value: "gitlab",
    label: "GitLab",
  },
  {
    value: "gitlab_enterprise",
    label: "GitLab Enterprise",
  },
  {
    value: "bitbucket",
    label: "Bitbucket",
  },
  {
    value: "bitbucket_server",
    label: "Bitbucket Server",
  },
  // {
  //   value: "other",
  //   label: "Other",
  // }
];

export const BITBUCKET_SERVER_DEFAULT_HTTP_PORT = "7990";
export const BITBUCKET_SERVER_DEFAULT_SSH_PORT = "7999";

export const IconWrapper = styled.div`
  height: 30px;
  width: 30px;
  background-position: center;
  background-size: contain;
  background-repeat: no-repeat;
  background-color: #ffffff;
  z-index: 1;
`;
