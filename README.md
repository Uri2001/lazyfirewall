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

Note: when running with `sudo`, the default config path becomes `/root/.config/lazyfirewall/config.toml`.  
Use `LAZYFIREWALL_CONFIG` if you want to keep a config in your user home.

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
- Live logs (firewalld/iptables)

## Phase status
- Phase 1–4 complete
- Phase 5 features added (templates, search, backups, panic mode, IPSets, logs)
- Config support: `config.toml` with `default_permanent` and `log_level`

## Keybindings
**Global**
- `?` help, `q`/`Ctrl+C` quit
- `--dry-run/-n` start in dry‑run mode

**Navigation**
- `Tab` switch focus, `j/k` move selection
- `1-6` switch tabs, `h/l` prev/next tab

**View**
- `P` toggle runtime/permanent
- `S` split diff view
- `L` live logs (firewalld/iptables)
- `r` refresh

**Zones**
- `n` new zone
- `d` delete zone
- `D` set default zone

**Main panel actions**
- `a` add service/port/rule/etc (contextual)
- `d` remove selected item
- `e` edit rich rule
- `m` toggle masquerade
- `i` add interface
- `s` add source
- `Enter` service details

**Runtime**
- `c` commit runtime → permanent
- `u` reload (revert runtime)

**Templates & backups**
- `t` apply template
- `Ctrl+R` backup restore menu
- `Ctrl+B` create backup

**Import/Export**
- `Ctrl+E` export zone (JSON/XML)
- `Alt+I` import zone (JSON/XML)

**Undo/Redo**
- `Ctrl+Z` undo
- `Ctrl+Y` redo

**Panic mode**
- `Alt+P` panic mode (type `YES`)

**IPSets**
- `n` new ipset (permanent)
- `a` add entry
- `d` remove entry
- `D` delete ipset

**Search**
- `/` search
- `n/N` next/prev match

**Input helpers**
- `Tab` autocomplete (export/import paths, service names)

## Backup location
`~/.config/lazyfirewall/backups`  
Backups are created automatically before the first mutation per zone, and can also be created manually.

## Notes
- When running with `sudo`, the default config path is `/root/.config/lazyfirewall/config.toml`.  
  Use `LAZYFIREWALL_CONFIG` to force a config from your user home.
- Logs default to `~/.config/lazyfirewall/lazyfirewall.log` unless `LAZYFIREWALL_LOG_STDERR=1`.
