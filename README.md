# trigger

A lightweight DevSecOps CLI utility for creating, managing, and running "triggers" — named commands that can be saved, reused, and executed with dynamic arguments and embedded scripts.

NOTICE: XDXD TOOL COMPLETELY Very Inefficient But Entertaining CODED. I'm gonna refactor it later. I just want some tool like that.

## Features

### Core Capabilities

- **Create Triggers** — Register named commands with optional default arguments and save them to disk
- **List Triggers** — View all registered triggers and their commands
- **Run Triggers** — Execute triggers by name with:
  - **Argument Substitution** — Use `[arg0]`, `[arg1]`, etc. as placeholders for runtime arguments
  - **Payload Piping** — Pass JSON/text payloads via stdin
  - **Timeout Support** — Kill long-running commands
  - **Dry-run Mode** — Preview what would execute without running
- **Script Embedding** — Automatically detect and embed script files (`.sh`, `.py`, `.js`, `.rb`, `.php`, `.pl`, `.lua`, `.groovy`, `.swift`) so they don't need to exist on disk
- **Sign Payloads** — Compute HMAC-SHA256 signatures using `TRIGGER_SECRET` env var (useful for webhook verification)
- **Shell Completions** — Generate completion scripts for bash, zsh, fish, and powershell

## Installation

### Build from Source

```bash
git clone https://github.com/steugen/trigger.git
cd trigger
go build -o trigger main.go
```

## Configuration

Triggers are stored in `$XDG_CONFIG_HOME/trigger/triggers.json` (or `~/.config/trigger/triggers.json` on most systems).

Scripts are embedded and stored in `~/.config/trigger/scripts/`.

## Usage

### Creating Triggers

#### Basic Trigger

Create a simple trigger that runs a command:

```bash
trigger create mycommand -- echo "Hello, World!"
```

### Deleting Triggers

Delete a trigger with name: 

```bash
trigger delete --name mycommand 
```

#### Trigger with Arguments

Create a trigger with placeholder arguments that can be filled at runtime:

```bash
trigger create backup -- tar -czf '[arg0]' /etc
```

When running, you can provide arguments:

```bash
trigger run --name backup --args ./backup.tar.gz
# Executes: tar -czf ./backup.tar.gz /etc
```

Multiple argument placeholders:

```bash
trigger create copy -- cp '[arg0]' '[arg1]'
```

```bash
trigger run --name copy --args /source/file.txt /dest/file.txt
# Executes: cp /source/file.txt /dest/file.txt
```

**Note:** Quote the placeholder arguments (`[arg0]`, `[arg1]`, etc.) to prevent your shell from interpreting the square brackets as glob patterns.

#### Trigger with Embedded Script

Create a trigger from a script file. The script content is automatically embedded, so you don't need to have the script present when running the trigger:

```bash
trigger create alert-slack -- ./send_slack_alert.sh
```

This will:
- Read the content of `./send_slack_alert.sh`
- Embed it into the trigger configuration
- Store it in `~/.config/trigger/scripts/alert-slack.sh`

When you run it, the embedded script will be executed:

```bash
trigger run --name alert-slack --payload message.json
```

#### Combined: Script with Argument Placeholders

```bash
trigger create process-data -- ./transform.py '[arg0]' '[arg1]'
```

```bash
trigger run --name process-data --args input.csv output.csv
# Executes: transform.py input.csv output.csv
```

### Listing Triggers

View all registered triggers:

```bash
trigger list
```

Output:
```
- backup: tar -czf [arg0] /etc
- alert-slack: /home/user/.config/trigger/scripts/alert-slack.sh [embedded: send_slack_alert.sh]
- copy: cp [arg0] [arg1]
```

### Running Triggers

#### Basic Execution

```bash
trigger run --name mycommand
```

#### With Arguments

```bash
trigger run --name backup --args ./backup.tar.gz
trigger run --name copy --args file1.txt file2.txt
```

#### With Payload File

Pipe a file to stdin (useful for webhooks, data processing):

```bash
trigger run --name alert-slack --payload event.json
```

The file contents will be written to the command's stdin.

#### With Timeout

Kill the command if it takes too long:

```bash
trigger run --name long-task --timeout 30s
```

#### Dry-run Mode

Preview what would execute without actually running it:

```bash
trigger run --name backup --args ./backup.tar.gz --dry-run
# Output: [dry-run] would run: tar -czf ./backup.tar.gz /etc
```

#### Verbose Output

```bash
trigger run --name mycommand --verbose
```

### Signing Payloads

Compute HMAC-SHA256 signatures for webhook verification or payload authentication:

Set the secret:

```bash
export TRIGGER_SECRET="your-secret-key"
```

Sign a payload file:

```bash
trigger sign --payload message.json
```

Sign from stdin:

```bash
echo '{"event":"push"}' | trigger sign
```

Output: hexadecimal HMAC-SHA256 digest

### Shell Completions

Generate completion scripts for your shell:

#### Bash

```bash
trigger completion bash > /etc/bash_completion.d/trigger
# or
trigger completion bash > ~/.bash_completion.d/trigger
source ~/.bash_completion.d/trigger
```

#### Zsh

```bash
trigger completion zsh > ~/.zsh/completions/_trigger
```

#### Fish

```bash
trigger completion fish > ~/.config/fish/completions/trigger.fish
```

#### PowerShell

```powershell
trigger completion powershell | Out-String | Invoke-Expression
```

## Examples

### 1. Database Backup Trigger

```bash
# Create a backup trigger with date argument
trigger create db-backup -- mysqldump -u root [arg0] > /backups/[arg0]-$(date +%Y%m%d).sql
trigger run --name db-backup --args mydb
```

### 2. Slack Notification Trigger

```bash
# Create script: send_slack.sh
#!/bin/bash
curl -X POST -H 'Content-type: application/json' \
  --data @- \
  https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Register it
trigger create notify-slack -- ./send_slack.sh

# Use it
trigger run --name notify-slack --payload alert.json
```

### 3. Log Processing Trigger

```bash
trigger create process-logs -- gawk -f '[arg0]' '[arg1]'
trigger run --name process-logs --args filter.awk access.log
```

### 4. Webhook Handler with Signature Verification

```bash
# Create a webhook handler script
trigger create webhook-handler -- ./verify_and_process.sh

# Sign incoming webhook
trigger sign --payload webhook_payload.json

# Run handler with payload
trigger run --name webhook-handler --payload webhook_payload.json
```

## Global Flags

```
--config string      Config file (optional)
--dry-run           Don't execute commands; show what would run
-v, --verbose       Verbose output
```

## Directory Structure

```
~/.config/trigger/
├── triggers.json      # All registered triggers
└── scripts/           # Embedded script files
    ├── alert-slack.sh
    ├── process-data.py
    └── ...
```

## Architecture

### Trigger Storage Format

Each trigger is stored in `triggers.json`:

```json
{
  "name": "backup",
  "command": "tar",
  "args": ["-czf", "[arg0]", "/etc"],
  "script_content": "",
  "script_path": "",
  "created_at": "2024-01-15T10:30:00Z"
}
```

For embedded scripts:

```json
{
  "name": "alert-slack",
  "command": "/home/user/.config/trigger/scripts/alert-slack.sh",
  "args": [],
  "script_content": "#!/bin/bash\ncurl ...",
  "script_path": "send_slack_alert.sh",
  "created_at": "2024-01-15T10:35:00Z"
}
```

### Script Detection

Scripts are identified by file extension. Supported extensions:
- `.sh`, `.py`, `.js`, `.rb`, `.php`, `.pl`, `.lua`, `.groovy`, `.swift`

When a script file is detected during trigger creation, its content is:
1. Read from disk
2. Embedded into the trigger JSON
3. Written to `~/.config/trigger/scripts/` for execution

This approach allows triggers to be portable — the script doesn't need to exist at the original path when running.

## Argument Placeholder System

Placeholders use the format `[argN]` where `N` is a zero-indexed number:

- `[arg0]` — First runtime argument
- `[arg1]` — Second runtime argument
- `[arg2]` — Third runtime argument
- etc.

Example:

```bash
trigger create deploy -- rsync -av '[arg0]' '[arg1]'
trigger run --name deploy --args ./src/ user@server:/dest/
```

This resolves to: `rsync -av ./src/ user@server:/dest/`

## Environment Variables

- `TRIGGER_SECRET` — Used by the `sign` command to compute HMAC-SHA256
- `XDG_CONFIG_HOME` — Config directory (defaults to `~/.config` if not set)

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
