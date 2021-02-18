# Allow applying application configuration from CLI as a stand-alone operation

Admin Console CLI has a way to specify appication configuration during the initial install.
However there is no dedicated way to use CLI to update application configuration.
`kots download` and `kots upload` commands can be used to achieve this goal.
However, this does not allow to easily patch the config without knowing the current set of keys and values.

## Goals

- Configuration can be applied independently from the rest of the app spec (unlike kots download/upload commands).
- Configuration can be applied to new versions of the app before they are deployed.
- Individual config keys can be added, modified, and removed.

## Non Goals

- Changing current architecture, in which config changes are always separate from other changes, such as license updates or application version updates.
License updates and upstream updates will still create their own versions with existing config.
- Covering every add/delete/modify config key cases in convenience commands, though the CLI can be extended later.
- Interractive editing in terminal.
- Adding the ability to make config changes to a deployed application without having to also take the latest downloaded upstream app version. 

## Background

Some application require periodic configuration updates.
CLI will provide a way to automate such operations.

## High-Level Design

There will be two modes of operation:
1. Apply an entire configuration to the app.
1. Apply a config change to a subset of values.  This is mostly a convenience method since this can be achieved with the first mode.

## Detailed Design

The following command will be added to CLI:

```
kubectl kots set config [options] <app-slug> [KEY_1=VAL_1 ... KEY_2=VAL_2]
```

The following flags will be supported, in addition to all higher level flags:

```
 --key
 --value
 --from-file <path to config file>
 --merge
 --deploy
 --skip-preflights
 ```

 | Flag | Description |
| :---- | ----------- |
| `--key` | This is the name of the config key whose value will be replaced.  Either `--value` or `--from-file` is required with this parameter. |
| `--value` | This is the new value of the config option named by the `--key` parameter. In case of a secret, this is the clear text value. |
| `--from-file` | With `--key`, this is a path to a text file that contains the value for the specified key.  Without `--key`, this is a path to a config file compiant with KOT's `kind: ConfigValues` format.  By default, the contents of this file will be used to replace existing app config in its entirety.  This property can be used to delete config items that are no longer used. |
| `--merge` | When this parameter is specified, only the keys included in the `--from-file` file will be replaced and all other values will be preserved.  This can be used to avoid creating multiple app versions when setting one value at a time when more than one config value needs to be changed. |
| `--deploy` | By default the new app version will be created but not deployed.  This parameter has the same function as the one in `kots upstream upgrade` command. |
| `--skip-preflights` | By default preflight checks will run on config update. This parameter has the same function as the one in `kots upstream upgrade` command. |

### Examples

```
kubectl kots set config myapp --from-file /path/to/local/config.yaml
```

```
kubectl kots set config myapp --key config-item-name --from-file /path/to/config/file/value.txt
```

```
kubectl kots set config myapp config-item-name="config item value"
```

```
kubectl kots set config myapp --key config-item-name --value "config item value"
```

## Alternatives Considered

None.

## Security Considerations

Config values specified on the command line are always clear text and can potentially expose secrets.  Standard practices should be followed to avoid this: disabling shell history, using environent variables, using files as input.
