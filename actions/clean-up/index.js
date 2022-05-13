import { getExecOutput, exec } from '@actions/exec'
import { getInput } from '@actions/core'
const tests = [
  {
    backend_config: "embedded-airgapped-upgrade-backend-config.tfvars",
    terraform_script: "embedded-airgapped-upgrade.sh",
  },
  {
    backend_config: "embedded-airgapped-install-backend-config.tfvars",
    terraform_script: "embedded-airgapped-install.sh"
  },
  {
    backend_config: "embedded-airgapped-install-backend-config-k3s.tfvars",
    terraform_script: "embedded-airgapped-install-k3s.sh",
    kubernetes_distro: "k3s"
  },
  {
    backend_config: "embedded-online-install-backend-config.tfvars",
    terraform_script: "embedded-online-install.sh"
  },
  {
    backend_config: "embedded-online-upgrade-backend-config.tfvars",
    terraform_script: "embedded-online-upgrade.sh",
  },
  {
    backend_config: "existing-airgapped-install-admin-backend-config.tfvars",
    terraform_script: "existing-airgapped-install-admin.sh"
  },
  {
    backend_config: "existing-airgapped-install-minimum-backend-config.tfvars",
    terraform_script: "existing-airgapped-install-minimum.sh"
  },
  {
    backend_config: "existing-online-upgrade-admin-backend-config.tfvars",
    terraform_script: "existing-online-upgrade-admin.sh",
  },
  {
    backend_config: "existing-online-upgrade-minimum-backend-config.tfvars",
    terraform_script: "existing-online-upgrade-minimum.sh",
  },
  {
    backend_config: "existing-online-install-admin-backend-config.tfvars",
    terraform_script: "existing-online-install-admin.sh"
  },
  {
    backend_config: "existing-online-install-minimum-backend-config.tfvars",
    terraform_script: "existing-online-install-minimum.sh"
  },
  {
    backend_config: "existing-airgapped-upgrade-admin-backend-config.tfvars",
    terraform_script: "existing-airgapped-upgrade-admin.sh",
  },
  {
    backend_config: "existing-airgapped-upgrade-minimum-backend-config.tfvars",
    terraform_script: "existing-airgapped-upgrade-minimum.sh",
  }
];

const awsConfig = {
  AWS_DEFAULT_REGION: getInput('AWS_DEFAULT_REGION'),
  AWS_ACCESS_KEY_ID: getInput('AWS_ACCESS_KEY_ID'),
  AWS_SECRET_ACCESS_KEY: getInput('AWS_SECRET_ACCESS_KEY')
}
await exec('terraform', ['init'], {
  env: awsConfig,
  cwd: 'automation/jumpbox'
});
const { stdout: workspaceOutput } = await getExecOutput('terraform', ['workspace', 'list'], {
  env: awsConfig,
  cwd: 'automation/jumpbox'
})
const automationWorkspaces = workspaceOutput.match(/automation-.*/g);

if(!automationWorkspaces) {
  process.exit(0);
}

for(const automationWorkspace of automationWorkspaces) {
  const { stdout: completionTimeRaw } = await getExecOutput(
    'terraform', ['output', '-raw', 'completion_timestamp'],
    {
      env: {
        ... awsConfig,
        TF_WORKSPACE: automationWorkspace
      },
      ignoreReturnCode: true,
      cwd: 'automation/jumpbox'
    });

  if(completionTimeRaw) {
    const currentTime = new Date();
    const completionTime = new Date(completionTimeRaw);
    // if(currentTime.getTime() - completionTime.getTime() > (1000 * 60 * 60 * 24)) {
    if(currentTime.getTime() - completionTime.getTime() > (1)) {
      for (const test of tests) {
        await exec('terraform', ['init', '-backend-config', test.backend_config, '-reconfigure'], {
          cwd: 'automation/cluster',
          env: awsConfig
        });
        await exec(test.terraform_script, ['destroy'], {
          cwd: 'automation/cluster',
          env: {
            ...awsConfig,
            TF_WORKSPACE: automationWorkspace,
          },
        });
        await exec('terraform', ['workspace', 'delete', automationWorkspace], {
          cwd: 'automation/cluster',
          env: {
            ... awsConfig,
            TF_WORKSPACE: 'default'
          },
        })
      }
      await exec('terraform', ['destroy', '-auto-approve'], {
        cwd: 'automation/jumpbox',
        env: {
          ...awsConfig,
          TF_WORKSPACE: automationWorkspace,
        },
      });
    }

    await exec('terraform', ['workspace', 'delete', automationWorkspace], {
      env: {
        ... awsConfig,
        TF_WORKSPACE: 'default'
      },
      cwd: 'automation/jumpbox'
    })
  }
}