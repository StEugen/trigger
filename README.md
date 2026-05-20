# trigger

A lightweight CLI for storing, listing, and running reusable local commands as named triggers.

## Features

- Create named triggers and store them locally
- Run triggers with runtime placeholders like `[arg0]`, `[arg1]`, and so on
- Pass a file to trigger stdin with `--payload`
- Kill long-running commands with `--timeout`
- Preview execution with `--dry-run`
- Embed local scripts into trigger storage automatically
- Store raw shell command lines for pipelines and redirection with shell triggers
- Use an interactive TUI to create, run, and inspect triggers
- Open a constrained TUI shell backed by `whitelist_shell.json`
- Compute HMAC-SHA256 signatures with `trigger sign`
- Generate shell completions for `bash`, `zsh`, `fish`, and `powershell`

## Installation

### Build from source

```bash
git clone https://github.com/steugen/trigger.git
cd trigger
go build -o trigger ./main.go
```

## Configuration

By default, trigger stores data under:

- `$XDG_CONFIG_HOME/trigger` when `XDG_CONFIG_HOME` is set
- `~/.config/trigger` otherwise

Files created there:

- `triggers.json`: JSON array of registered triggers
- `scripts/`: embedded script copies used at runtime
- `whitelist_shell.json`: allowlist for the TUI shell

Default `whitelist_shell.json`:

```json
{
  "commands": ["cd", "ls"]
}
```

## Commands

### Create a direct trigger

```bash
trigger create backup -- tar -czf '[arg0]' /etc
```

This stores a direct command plus argv. It does not use a shell.

### Create a shell trigger

Use this for pipelines, redirection, command substitution, or other shell syntax:

```bash
trigger create db-dump --shell -- "pg_dump [arg0] | zstd > [arg1]"
```

Shell trigger placeholders are shell-escaped before execution.

### Create a trigger from a script

```bash
trigger create notify-slack -- ./send_slack.sh
```

If the command points to an existing script file with a supported extension, trigger embeds its contents and writes a runnable copy into the config `scripts/` directory.

Supported script extensions:

- `.sh`
- `.py`
- `.js`
- `.rb`
- `.php`
- `.pl`
- `.lua`
- `.groovy`
- `.swift`

### List triggers

```bash
trigger list
```

Example output:

```text
- backup: tar [-czf [arg0] /etc]
- notify-slack: /home/user/.config/trigger/scripts/notify-slack.sh [] [embedded: send_slack.sh]
- db-dump: [shell] pg_dump [arg0] | zstd > [arg1]
```

### Run a trigger

```bash
trigger run --name backup --args ./backup.tar.gz
trigger run --name db-dump --args mydb dumps/mydb.sql.zst
```

Runtime arguments can be passed through `--args` and also as extra positional args after the flags.

With stdin payload:

```bash
trigger run --name notify-slack --payload alert.json
```

With timeout:

```bash
trigger run --name long-task --timeout 30s
```

Dry run:

```bash
trigger run --name backup --args ./backup.tar.gz --dry-run
```

Verbose mode:

```bash
trigger run --name backup --args ./backup.tar.gz --verbose
```

### Delete a trigger

```bash
trigger delete --name backup
```

If the trigger has an embedded script, the stored script file is removed too.

### Sign a payload

Set the secret:

```bash
export TRIGGER_SECRET="your-secret-key"
```

Sign a file:

```bash
trigger sign --payload message.json
```

Sign stdin:

```bash
echo '{"event":"push"}' | trigger sign
```

### Open the TUI

```bash
trigger tui
```

The TUI can:

- list triggers
- create triggers from prompted command lines
- run existing triggers
- open a constrained shell

TUI shell behavior:

- only commands listed in `whitelist_shell.json` can run
- `cd` is handled internally
- `ls` is allowed by default
- this shell executes direct commands only; it does not interpret pipelines or redirection

When you create a trigger in the TUI, common shell operators such as `|`, `>`, `<`, `;`, and `&` are detected and the trigger is stored as a shell trigger automatically.

### Generate completions

```bash
trigger completion bash
trigger completion zsh
trigger completion fish
trigger completion powershell
```

### Print the version

```bash
trigger version
```

## Global flags

Available on all commands:

```text
--dry-run       don't execute commands; show what would run
-v, --verbose   verbose output
```

## Examples

### Database backup

```bash
trigger create db-backup --shell -- "pg_dump [arg0] | zstd > [arg1]"
trigger run --name db-backup --args mydb dumps/mydb.sql.zst
```

### Slack notification

```bash
trigger create notify-slack -- ./send_slack.sh
trigger run --name notify-slack --payload alert.json
```

### Log processing

```bash
trigger create process-logs -- gawk -f '[arg0]' '[arg1]'
trigger run --name process-logs --args filter.awk access.log
```

### Deploy copy

```bash
trigger create deploy -- rsync -av '[arg0]' '[arg1]'
trigger run --name deploy --args ./src/ user@server:/dest/
```

## Storage format

`triggers.json` is a JSON array. Direct triggers look like this:

```json
[
  {
    "name": "backup",
    "command": "tar",
    "args": ["-czf", "[arg0]", "/etc"],
    "created_at": "2024-01-15T10:30:00Z"
  }
]
```

Shell triggers add `shell` and `command_line`:

```json
[
  {
    "name": "db-dump",
    "command": "sh",
    "command_line": "pg_dump [arg0] | zstd > [arg1]",
    "shell": true,
    "created_at": "2024-01-15T10:35:00Z"
  }
]
```

Embedded script triggers include script metadata and stored content:

```json
[
  {
    "name": "notify-slack",
    "command": "/home/user/.config/trigger/scripts/notify-slack.sh",
    "script_content": "#!/bin/sh\ncurl ...",
    "script_path": "send_slack.sh",
    "created_at": "2024-01-15T10:40:00Z"
  }
]
```

## Environment variables

- `TRIGGER_SECRET`: secret used by `trigger sign`
- `XDG_CONFIG_HOME`: base directory for trigger config files

## License

See [LICENSE](LICENSE).
