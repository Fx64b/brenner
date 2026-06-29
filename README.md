# 🔥 Brenner

> Burn ISO images onto USB drives from your terminal — like burning music to a CD.

**Brenner** is German for *burner*. The name is a deliberate nod to the old
ritual of burning your favourite music and films onto a CD: you picked the
disc, you picked what went on it, you watched the little progress bar crawl to
100 %, and then you held something real in your hand. Brenner brings that same
small, satisfying ceremony to USB drives — pick a stick, pick an image, watch
it burn, done.

It's a single ~10 MB native binary. No Electron, no 200 MB GUI, no background
service. Just a fast, friendly terminal tool that also works beautifully over
SSH.

```
🔥 brenner
burn ISOs to USB — like music to a CD

┌ Select a USB device ─────────────────────────────┐
│ > [8.0 GB]   /dev/sdb — Samsung Flash Drive       │
│   [16.0 GB]  /dev/sdc — Kingston DataTraveler     │
└───────────────────────────────────────────────────┘

Writing archlinux-2026.06.01-x86_64.iso → /dev/sdb
████████████████░░░░░░░░  67%   412 MB/s   ETA 4s
820.1 MB / 1.2 GB

Write complete.
Verified OK.
```

## Features

- **Interactive TUI** — a guided device → action → image flow built with
  [huh](https://github.com/charmbracelet/huh) and a warm "ember" theme.
- **Scriptable flag mode** — `brenner flash --device … --iso … --yes` skips
  the UI entirely, so it composes with shell scripts.
- **Write + verify** — re-reads the device and compares a SHA-256 against the
  source image (disable with `--no-verify`).
- **Per-user ISO cache** — optionally stash images in `~/.brenner/` so your
  favourites are one keystroke away next time.
- **Wipe, three ways** — `quick` (default) erases the partition table +
  filesystem signatures in ~1 second so the drive is instantly reusable;
  `zero` overwrites every byte; `secure` overwrites with random data. See
  [Wiping a drive](#wiping-a-drive) for the speed/security trade-offs.
- **Steady writes** — on Linux, writes trigger incremental writeback
  (`sync_file_range`) so the page cache stays bounded; no flying-to-100%-then-
  stalling, and your desktop doesn't freeze mid-flash.
- **Works over SSH** — it's just a terminal program.

## Install

```sh
go install github.com/fx64b/brenner@latest
```

This drops a `brenner` binary in your `$GOPATH/bin` (usually `~/go/bin`).
Requires Go 1.24+.

Or download a pre-built binary from the
[Releases](https://github.com/fx64b/brenner/releases) page. An AUR package and a
Homebrew tap are on the roadmap.

## Usage

### Interactive

Run it with no arguments and follow the prompts:

```sh
brenner
```

It enumerates your removable drives, asks whether to **flash** or **wipe**,
lets you choose a cached ISO / scan your home directory / type a path, confirms
the destructive step, then writes and verifies.

### Non-interactive (flag mode)

```sh
# Flash an image (asks for confirmation)
brenner flash --device /dev/sdb --iso ~/Downloads/arch.iso

# Same, but skip the prompt and the verification step
brenner flash --device /dev/sdb --iso ~/Downloads/arch.iso --yes --no-verify

# Wipe a drive (quick erase by default; --mode zero|secure for a full overwrite)
brenner wipe --device /dev/sdb --yes
brenner wipe --device /dev/sdb --mode zero --yes

# List removable devices as a table (no root needed)
brenner list
```

If you leave out `--device`/`--iso`, Brenner falls back to the interactive flow.

> **Tip:** the target doesn't have to be a block device — point `--device` at a
> path like `./disk.img` and Brenner happily writes a flashable image file.

### Permissions

Writing to a real block device requires elevated privileges — but you don't
have to remember that. When Brenner is about to write and isn't running as root,
it re-launches itself through `sudo` and lets your system prompt for the
password. Read-only commands like `brenner list` never ask for it.

- **Linux / macOS:** Brenner escalates via `sudo` automatically (run as root to
  skip the prompt). The password prompt appears only after you've confirmed the
  write.
- **Windows:** run the terminal **as Administrator**.

## The ISO cache

After a successful flash, Brenner offers to copy the image into `~/.brenner/`
(skipped in scripts and when the image is already cached). Cached images appear
at the top of the picker next time. It's just a directory of `.iso` files — no
database — so you can drop files in or delete them by hand.

Override the location with the `BRENNER_HOME` environment variable. When you run
Brenner with `sudo`, it resolves `~` to the **invoking** user's home (via
`$SUDO_USER`) rather than `/root` — so the cache and the "scan home" feature work
on your own files; if that can't be determined it scans `/home`.

## Wiping a drive

Overwriting a whole USB stick is bound by its (often slow) flash write speed —
a 64 GB stick with no TRIM support takes ~1 hour to fully zero, and no amount of
software cleverness changes that. So Brenner offers three modes; pick based on
what you actually need:

| Mode | What it does | Speed (64 GB stick) | Recoverability |
|------|--------------|---------------------|----------------|
| `quick` *(default)* | Zeros the partition table + filesystem signatures (first/last 8 MB) so the OS sees a blank device; uses TRIM first when supported | **~1 second** | File data physically remains and is recoverable with forensic tools |
| `zero` | Overwrites every byte with `0x00` (TRIM / hardware-zeroing fast path on Linux when available) | Device-speed (**~1 h** without TRIM) | Gone by normal means |
| `secure` | Overwrites the whole device with random data | Device-speed (**~1 h+**) | Best-effort only — flash wear-leveling and over-provisioning mean overwrites can't reach every physical cell ([NIST SP 800-88](https://csrc.nist.gov/pubs/sp/800/88/r1/final) recommends the device's own sanitize command for flash, which USB sticks rarely expose) |

**Reclaiming a stick to reuse it? `quick` is all you need.** Only reach for
`zero`/`secure` when the data must actually be unrecoverable before the drive
leaves your hands — and know that on flash, even those are imperfect.

```sh
brenner wipe --device /dev/sdb                 # quick (interactive confirm)
brenner wipe --device /dev/sdb --mode zero -y  # full zero-fill
brenner wipe --device /dev/sdb --mode secure   # random overwrite
```

## Platform support

| Platform | Status | How devices are found |
|----------|--------|-----------------------|
| Linux    | Full, tested | `sysfs` (`/sys/block/*/removable`) + `syscall.Unmount` |
| macOS    | Best-effort | `diskutil list external` |
| Windows  | Best-effort | PowerShell `Get-Disk` (USB bus) |

The Linux path is the one developed and tested on a Linux host. The macOS and
Windows backends are implemented behind build tags and compile cleanly, but have
not yet been exercised on real hardware — feedback and PRs welcome.

## Demo without a USB stick

Set `BRENNER_FAKE_DEVICES=1` to populate the UI with a couple of sample drives —
handy for screenshots, demos, or trying the flow safely:

```sh
BRENNER_FAKE_DEVICES=1 brenner list
```

## Development

```sh
make build        # build ./brenner with version info
make test         # run the test suite
make vet          # go vet
make fmt-check    # verify gofmt cleanliness
make cross        # cross-compile checks (darwin, windows)
make help         # list all targets
```

CI (GitHub Actions) runs gofmt, `go vet`, `go test -race`, and the cross-compile
checks on every push and PR. Tagging `vX.Y.Z` triggers a GoReleaser build that
publishes binaries to the Releases page.

### Project layout

```
brenner/
├── cmd/                 cobra commands (root, flash, wipe, list) + shared exec helpers
└── internal/
    ├── device/          removable-device enumeration (Linux + darwin/windows stubs)
    ├── flash/           Copy / Verify / Wipe — the byte-pushing core
    ├── cache/           the ~/.brenner ISO store
    └── ui/              lipgloss theme, huh forms, animated progress bar
```

## A word of caution

Flashing and wiping **destroy everything** on the target device. Brenner
confirms before it writes and always shows you the device path and size — but
double-check that path. There is no undo on a burned disc, and there is none
here either.

## License

[MIT](./LICENSE) © fx64b
