import { getExecOutput, exec } from '@actions/exec'

const tests = [
  {
    name: "type=embedded cluster, env=airgapped, phase=upgraded install, rbac=cluster admin",
    backend_config: "embedded-airgapped-upgrade-backend-config.tfvars",
    terraform_script: "embedded-airgapped-upgrade.sh",
  },
  {
    name: "type=embedded cluster, env=airgapped, phase=new install, rbac=cluster admin",
    backend_config: "embedded-airgapped-install-backend-config.tfvars",
    terraform_script: "embedded-airgapped-install.sh"
  },
  {
    name: "type=embedded cluster, env=airgapped, phase=new install, rbac=cluster admin",
    backend_config: "embedded-airgapped-install-backend-config-k3s.tfvars",
    terraform_script: "embedded-airgapped-install-k3s.sh",
    kubernetes_distro: "k3s"
  },
  {
    name: "type=embedded cluster, env=online, phase=new install, rbac=cluster admin",
    backend_config: "embedded-online-install-backend-config.tfvars",
    terraform_script: "embedded-online-install.sh"
  },
  {
    name: "type=embedded cluster, env=online, phase=upgraded install, rbac=cluster admin",
    backend_config: "embedded-online-upgrade-backend-config.tfvars",
    terraform_script: "embedded-online-upgrade.sh",
  },
  {
    name: "type=existing cluster, env=airgapped, phase=new install, rbac=cluster admin",
    backend_config: "existing-airgapped-install-admin-backend-config.tfvars",
    terraform_script: "existing-airgapped-install-admin.sh"
  },
  {
    name: "type=existing cluster, env=airgapped, phase=new install, rbac=minimal rbac",
    backend_config: "existing-airgapped-install-minimum-backend-config.tfvars",
    terraform_script: "existing-airgapped-install-minimum.sh"
  },
  {
    name: "type=existing cluster, env=online, phase=upgraded install, rbac=cluster admin",
    backend_config: "existing-online-upgrade-admin-backend-config.tfvars",
    terraform_script: "existing-online-upgrade-admin.sh",
  },
  {
    name: "type=existing cluster, env=online, phase=upgraded install, rbac=minimal rbac",
    backend_config: "existing-online-upgrade-minimum-backend-config.tfvars",
    terraform_script: "existing-online-upgrade-minimum.sh",
  },
  {
    name: "type=existing cluster, env=online, phase=new install, rbac=cluster admin",
    backend_config: "existing-online-install-admin-backend-config.tfvars",
    terraform_script: "existing-online-install-admin.sh"
  },
  {
    name: "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
    backend_config: "existing-online-install-minimum-backend-config.tfvars",
    terraform_script: "existing-online-install-minimum.sh"
  },
  {
    name: "type=existing cluster, env=airgapped, phase=upgraded install, rbac=cluster admin",
    backend_config: "existing-airgapped-upgrade-admin-backend-config.tfvars",
    terraform_script: "existing-airgapped-upgrade-admin.sh",
  },
  {
    name: "type=existing cluster, env=airgapped, phase=upgraded install, rbac=minimal rbac",
    backend_config: "existing-airgapped-upgrade-minimum-backend-config.tfvars",
    terraform_script: "existing-airgapped-upgrade-minimum.sh",
  }
];
const workspaceOutput = await getExecOutput('terraform', ['workspace', 'list'], { cwd: 'automation/jumpbox' })
const automationWorkspaces = workspaceOutput.match(/automation-.*/g);

for(const automationWorkspace of automationWorkspaces) {
  exec('terraform', [ 'init' ], { cwd: 'automation/jumpbox' });
  const { stdout: completionTimestamp } = await getExecOutput(
    'terraform', ['output', 'completion_timestamp'],
    {
      env: {
        TF_WORKSPACE: automationWorkspace
      },
      cwd: 'automation/jumpbox'
    });
  const completionTime = new Date(completionTimestamp);
  const currentTime = new Date();
  if(currentTime.getTime() - completionTime.getTime() > (1000 * 60 * 60 * 24)) {
    for(const test of tests) {
      exec('terraform', [ 'init', '-backend-config', test.backend_config, '-reconfigure' ], { cwd: 'automation/cluster' });
      exec(test.terraform_script, [ 'destroy' ], {
        cwd: 'automation/cluster',
        env: {
          TF_WORKSPACE: automationWorkspace
        },
      });
    }
    exec('terraform', [ 'destroy', '-auto-approve' ], {
      cwd: 'automation/jumpbox',
      env: {
        TF_WORKSPACE: automationWorkspace
      },
    });
  }
}