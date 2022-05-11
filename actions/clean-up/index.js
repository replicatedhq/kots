import core from '@actions/core';
import exec from '@actions/exec'

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
const workspaceOutput = await executeWithOutput('terraform', ['workspace', 'list'], { cwd: 'kots-regression-automation/jumpbox' })
const automationWorkspaces = workspaceOutput.match(/automation-.*/g);

for(const automationWorkspace of automationWorkspaces) {
  exec.exec('terraform', [ 'init' ], { cwd: 'kots-regression-automation/jumpbox' });
  const { output: completionTimestamp } = await executeWithOutput(
    'terraform', ['output', 'completion_timestamp'],
    {
      env: {
        TF_WORKSPACE: automationWorkspace
      },
      cwd: 'kots-regression-automation/jumpbox'
    });
  const completionTime = new Date(completionTimestamp);
  const currentTime = new Date();
  if(currentTime.getTime() - completionTime.getTime() > (1000 * 60 * 60 * 24)) {
    for(const test of tests) {
      exec.exec('terraform', [ 'init', '-backend-config', test.backend_config, '-reconfigure' ], { cwd: 'kots-regression-automation/cluster' });
      exec.exec(test.terraform_script, [ 'destroy' ], {
        cwd: 'kots-regression-automation/cluster',
        env: {
          TF_WORKSPACE: automationWorkspace
        },
      });
    }
    exec.exec('terraform', [ 'destroy', '-auto-approve' ], {
      cwd: 'kots-regression-automation/jumpbox',
      env: {
        TF_WORKSPACE: automationWorkspace
      },
    });
  }

}

async function executeWithOutput(command, args, additionalOptions) {
  let output = '';
  let error = '';

  const options = {
    ... additionalOptions,
    listeners: {
      stdout: (data) => {
        output += data.toString();
      },
      stderr: (data) => {
        error += data.toString();
      }
    }
  }

  const exitCode = await exec.exec('terraform', ['workspace', 'list'], options);

  return {
    error,
    output,
    exitCode
  }
}