# Ridectl

## Overview
`ridectl` is Ridecell's internal tool that enables employees access and ways to interact with SummonPlatform instances. Employee's permissions are restricted to certain environments or features, depending on their role.

Some key features are:
1. shelling into an instance (`shell`)
2. shelling into an instance under the python environment (`pyshell`)
3. shelling into the instance's database (`dbshell`)
4. Obtaining dispatcher/support/reports account password (`password`)
5. Restart migrations for a summon instance(`restart-migrations`)
6. Restart all pods of a certain type (web|celeryd|etc) (`restart`)
e.g.
    a. Summon-platform
    `ridectl restart summontest-dev web`
    b. Microservice
    `ridectl restart svc-us-master-webhook-sms web`
7. Create new summon-platform instance yml (`new`)

For a full list of functionalities, run `ridectl --help`

## Installing `ridectl`

### Manual Installation
You can find pre-compiled macOS and Linux binaries for `ridectl` [on the GitHub releases page](https://github.com/Ridecell/ridectl/releases/latest).

Download the one appropriate to your platform, unzip it, and copy it to `/usr/local/bin/ridectl` or similar. Run `ridectl -h` to confirm it is installed correctly.

# Mac-os
```
curl -L "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_macos.zip" -o ./ridectl_macos.zip
unzip ridectl_macos.zip
chmod 0755 ridectl
cp ridectl /usr/local/bin/ridectl
ridectl -h
```
When running ridectl for first time, mac os will not allow the binary to execute. So solve this issue, navigate to System Prefrences > Security & Privacy and in General section, allow ridectl to open.
# Linux
```
curl -L "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_linux.zip" -o ./ridectl_linux.zip
unzip ridectl_linux.zip
chmod 0755 ridectl
cp ridectl /usr/local/bin/ridectl
ridectl -h
```

# Add kubernetes contexts
You can follow the quip doc [here](https://ridecell.quip.com/O8W1AaqtWWAH/Ridectl)

Note:
Old ridectl code is still present [here](https://github.com/Ridecell/ridectl/tree/ridectl-v0.0.0). Ref:- ridectl-v0.0.0