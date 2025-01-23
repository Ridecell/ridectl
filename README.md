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
### Linux
```
curl -L "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_linux.zip" -o ./ridectl_linux.zip
unzip ridectl_linux.zip
chmod 0755 ridectl
sudo cp ridectl /usr/local/bin/ridectl
```

## Environment variables

List of environment variables used by `ridectl`:

| Environment variable | Value | Description |
| -------------------- | ------------- | ---------------- |
| `RIDECTL_SKIP_UPGRADE` | `true\|false` | If set `true`, skips auto-upgrade of ridectl version; used in Github actions workflows |
| `RIDECTL_SKIP_AWS_SSO` | `true\|false` | If set `true`, ridectl uses default AWS configuration instead of AWS SSO; used in Github actions workflows |
| `EDITOR` | `vim`, `code`, etc | Sets editor's binary path for `ridectl edit` command |
| `RIDECTL_TSH_CHECK` | `true\|false` | If set `false`, ridectl does not check for tsh login profile; used in Github actions workflows |
