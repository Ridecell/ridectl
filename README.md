# Ridectl

## Overview
`ridectl` is Ridecell's internal tool that enables employees access and ways to interact with SummonPlatform/Microservice instances. Employee's permissions are restricted to certain environments or features, depending on their role.

Some key features are:
1. Shelling into an instance (`shell`)\
    a. Summon-platform
    ```
    ridectl shell summontest-dev
    ```
    b. Microservice
    ```
    ridectl shell svc-us-master-webhook-sms
    ```
2. Shelling into an instance under the python environment (`pyshell`)\
    a. Summon-platform
    ```
    ridectl pyshell summontest-dev
    ```
    b. Microservice
    ```
    ridectl pyshell svc-us-master-webhook-sms
    ```
3. Shelling into the instance's database (`dbshell`)\
    a. Summon-platform
    ```
    ridectl dbshell summontest-dev
    ```
    b. Microservice
    ```
    ridectl dbshell svc-us-master-webhook-sms
    ```
4. Obtaining supported password/connection details (`password`)\
    a. Summon-platform
    ```
    ridectl password summontest-dev
    ```
5. Restart migrations for a summon instance(`restart-migrations`)\
    a. Summon-platform
    ```
    ridectl restart-migrations summontest-dev
    ```
6. Restart all pods of a certain type (web|celeryd|etc) (`restart`)\
    a. Summon-platform
    ```
    ridectl restart summontest-dev web
    ```
    b. Microservice
    ```
    ridectl restart svc-us-master-webhook-sms web
    ```
For a full list of functionalities, run `ridectl --help`

## Installing `ridectl`

### Mac-os
```
curl -L "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_macos.zip" -o ./ridectl_macos.zip
unzip ridectl_macos.zip
chmod 0755 ridectl
sudo cp ridectl /usr/local/bin/ridectl
```
**Note:** When running `ridectl` for first time, Mac OS will not allow the binary to execute. So, to solve this issue, navigate to `System Prefrences` > `Security & Privacy` and in `General` section, allow `ridectl` to open.

### Upgrading `ridectl`

```
brew upgrade ridectl
```

### Linux
```
curl -L "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_linux.zip" -o ./ridectl_linux.zip
unzip ridectl_linux.zip
chmod 0755 ridectl
sudo cp ridectl /usr/local/bin/ridectl
```

## Prerequisites

Before using `ridectl`, ensure the following are set up:

1. **Teleport Login**: Login to teleport using:
   ```
   tsh login --proxy=teleport.aws-us-support.ridecell.io:443 --auth=local --user=<your-ridecell-email-id>
   ```
   Note: Teleport session expires after 12 hours. Re-login when prompted.

2. **AWS SSO Profile**: `ridectl` requires AWS KMS permissions for encrypt/decrypt operations. Set up AWS SSO profiles following the [Infra-Auth AWS guide](https://github.com/ridecell/infra-auth/wiki/Configuring-AWS).

3. **Editor**: Set your preferred editor for `ridectl edit`:
   ```
   export EDITOR=vim             # or: export EDITOR="code -w"  for VS Code
   ```

## Environment Variables

List of environment variables used by `ridectl`:

| Environment variable | Value | Description |
| -------------------- | ------------- | ------------------ |
| `RIDECTL_SKIP_UPGRADE` | `true\|false` | If set `true`, skips auto-upgrade of ridectl version; used in Github actions workflows |
| `RIDECTL_SKIP_AWS_SSO` | `true\|false` | If set `true`, ridectl uses default AWSconfiguration instead of AWS SSO; used in Github actions workflows |
| `EDITOR` | `vim`, `code`, etc | Sets editor's binary path for `ridectl edit` command |
| `RIDECTL_TSH_CHECK` | `true\|false` | If set `false`, ridectl does not check for tsh login profile; used in Github actions workflows |

## Local Development Setup

Use this section to build and test `ridectl` locally before creating a release.

### Prerequisites

- Go (1.21+ recommended)
- `wget` installed (`brew install wget` on macOS)
- `make` installed
- Access to GitHub Ridecell org

### Build locally

**macOS**:
```
make build_macos
```

**Linux**:
```
make build_linux
```

This will:
1. Download the `tsh` binary for the version defined by `TCH_VERSION` in the `Makefile`.
2. Embed it into the `pkg/exec/bin/` directory.
3. Build the final `ridectl` binary into `bin/`.

### Running your local build

To prevent auto-upgrade from overwriting your local build:
```
export RIDECTEÓKIP_UPGRADE=true
bin/ridectl.macos shell summontest-dev
```

### Pre-release verification checklist

Always verify the following commands work with your locally built binary before releasing:

- [ ] `ridectl edit` (used for kubernetes-summon)
- [ ] `ridectl postgresdumps` (used in kubernetes-microservices, custom-actions)
- [ ] `ridectl encrypt` / `ridectl decrypt` (used in svc-* repos' Makefiles)
- [ ] `ridectl pyshell`
- [ ] `ridectl dbshell`
- [ ] `ridectl password`
- [ ] `ridectl shell`

## AWS Configuration

`ridectl` requires AWS access for KMS operations (encrypt/decrypt).

**Option 1 - AWS SSO (recommended)**: Follow [Infra-Auth AWS SSO guide](https://github.com/ridecell/infra-auth/wiki/Configuring-AWS).

**Option 2 - AWS Access Keys**:
```
aws configure
```
Paste your Access Key and Secret Key when prompted. You can skip region.
If you don't have AWS keys, raise a ticket to #devops-engineers on Slack.

## FAQs & Troubleshooting

For common issues and resolutions, refer to the [Ridectl Runbook & FAQs](https://github.com/Ridecell/ridectl/blob/master/RUNBOOK.md).

Common issues:

- **`No $EDITOR set` error**: Set the `EDITOR` environment variable before using `ridectl edit`. Example: `export EDITOR=vim`
- **Failed to update binary error**: Run `ridectl` with `sudo` when upgrading.
- **Teleport login expired**: Re-run the `tsh login` command; sessions expire after 12 hours.
- **Decryption errors**: Ensure your  AWS KMS-grants profile access is set up correctly. Refer to the [Infra-Auth AWS guide](https://github.com/ridecell/infra-auth/wiki/Configuring-AWS).
- **`ssh: cert is not yet valid`**: Ensure your system time is correct and synced.
- **No valid cluster found**: Re-login to Teleport and refresh your kubeconfig contexts.

For additional help, drop a message on the `#devops-engineers` Slack channel.
