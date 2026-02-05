# LazyFirewall

LazyFirewall is a terminal UI for managing `firewalld` using its D‑Bus API (no `firewall-cmd` parsing).

## Requirements
- Linux with `firewalld` 1.0+
- Go 1.22+

## Build
```bash
make build
```

## Run
```bash
sudo ./lazyfirewall
```

Run without sudo for read‑only mode:
```bash
./lazyfirewall
```

Dry‑run mode (no changes applied):
```bash
sudo ./lazyfirewall --dry-run
```

Version:
```bash
./lazyfirewall --version
```

Log level:
```bash
./lazyfirewall --log-level debug
```

Disable colors:
```bash
./lazyfirewall --no-color
```

## Config file
Default path: `~/.config/lazyfirewall/config.toml`  
Override with: `LAZYFIREWALL_CONFIG=/path/to/config.toml`

Example:
```toml
[ui]
theme = "default"

[behavior]
default_permanent = true
auto_refresh_interval = 0

[advanced]
log_level = "info"
```

## Highlights
- Zones sidebar with active/default markers
- Tabs: Services, Ports, Rich Rules, Network, IPSets, Info
- Runtime/Permanent toggle (`P`) and split diff view (`S`)
- Templates, search/filter, service details
- Backup/restore, export/import, undo/redo
- Panic mode with safety confirmation
- IPSets list and entry management

## Keybindings (essentials)
- `Tab` switch focus, `j/k` move, `1-6` switch tabs, `h/l` prev/next tab
- `a` add, `d` delete, `D` default/delete (contextual)
- `P` toggle runtime/permanent, `S` split diff
- `c` commit runtime → permanent, `u` reload (revert runtime)
- `Ctrl+R` backup restore, `Ctrl+B` create backup
- `Ctrl+E` export, `Alt+I` import
- `Ctrl+Z / Ctrl+Y` undo / redo
- `Alt+P` panic mode
- `/` search

## Backup location
`~/.config/lazyfirewall/backups`  
Backups are created automatically before the first mutation per zone, and can also be created manually.
