# Security Policy

susfile opens and parses files it did not create — including deliberately hostile
ones. A defect that lets a crafted file crash it, hang it, exhaust memory, make
it touch a file the user did not name, or open a network connection is not a
cosmetic bug. Please treat this file as more than boilerplate.

## Reporting a vulnerability

**Do not open a public issue for a security problem.**

Report it privately through GitHub's private vulnerability reporting:

1. Go to <https://github.com/kawaiipantsu/susfile/security/advisories/new>
2. Describe the issue, the impact, and how to reproduce it.

If private reporting is unavailable, open a public issue that says only *"I have a
security report, please provide a private channel"* — no detail — and wait for a
maintainer to respond.

### What to include

- `susfile version` output and how the binary was built or installed.
- Operating system and architecture.
- The **smallest input file that triggers it**. If it is small, attach it
  (base64 in the report is fine). If it is large, attach a generator script.
- What an attacker gains and what access they need to start.
- A proof of concept if you have one.

### What to expect

susfile is a young project maintained by volunteers. We will acknowledge your
report, tell you whether we consider it in scope, and keep you informed. Fixes
ship in a release with a `Security` section in `CHANGELOG.md`. We credit you in
the advisory and changelog unless you ask us not to. If a report goes unanswered
for a month, disclose.

## Supported versions

susfile is pre-1.0. Only the latest release and the `develop` branch are
supported. There are no backports.

## What is in scope

- **Any outbound network connection.** susfile is entirely offline by design.
  There is no HTTP client in the module. A DNS lookup, a socket, a connection of
  any kind, triggered by any input or flag, is a vulnerability.
- **Memory exhaustion from a crafted file** — an input that makes susfile
  allocate far more than `--max-bytes` / the documented cap, e.g. a header that
  claims a huge section, an archive-like structure, a pathological string table.
- **Unbounded CPU / a hang** a remote-supplied file can trigger — a parser that
  does not terminate, quadratic scanning, an infinite loop on malformed input.
- **A panic** reachable from file content: an out-of-range slice while parsing
  ELF/PE/Mach-O, the magic table, the micro-block scan or string extraction.
- **Reading a file the user did not name** — path traversal through an argument,
  following a symlink into `/proc`, `/sys` or a device when `--allow-special`
  was not given, or TOCTOU between `stat` and open.
- **Writing anything.** susfile must never modify the analysed file or write
  outside an explicit `--json`/output path.
- Dependency vulnerabilities susfile actually reaches. `make security` runs
  `govulncheck`; CI runs it on every push and PR.

## What is out of scope

- **A user pointing susfile at a file and it getting analysed.** That is the
  entire purpose. The interesting question is whether a *malformed* file breaks
  the tool, not whether a valid one is read.
- **A slow but bounded analysis of a legitimately enormous file** within the
  cap. Use `--max-bytes` to bound it further.
- **The heuristic being wrong.** The block classifier and the verdict are
  best-effort guesses; a misclassification is a quality issue, filed as a normal
  bug, not a vulnerability.
- Vulnerabilities in Go's `debug/elf`, `debug/pe`, `debug/macho` or in
  `gabriel-vasile/mimetype` themselves — report those upstream. susfile's
  *handling* of a panic or bad output from them is in scope.
- Attacks requiring an attacker who already has your shell or can already write
  to the files you analyse.

## The security model

### susfile only reads, and only what you name

The single positional argument is the only file opened for analysis. It is
`stat`-ed first and refused unless it is a regular file (or `--allow-special` is
given explicitly). Nothing is written back. There is no config file, no cache,
no state directory.

### Bounded by construction

The full-buffer analysis (histogram, micro-block scan, string extraction) reads
at most `--max-bytes` (default documented in `PROJECT.md`); beyond that, blocks
are sampled rather than materialised. Hashing and the global entropy pass stream
in fixed-size chunks regardless of file size. Structural parsers
(`debug/elf|pe|macho`) are wrapped so a parse panic becomes a populated `Result`
with a `¯\_(ツ)_/¯` verdict, not a crash.

### Offline, enforced by absence

There is no `net/http` import in this module and CI keeps it that way. susfile
does no update checks, no hash-reputation lookups, no "is this file known"
queries, and sends no telemetry. Everything it reports is computed locally from
the bytes you gave it.

### Supply chain

`CGO_ENABLED=0` and `-trimpath` on every build. Dependencies are pinned through
Go modules and kept to four, all pure Go. Release archives ship with SHA-256
sums. `.deb` packages are currently **unsigned** — verify the archive checksums.

## Known limits

- The magic-signature table is a curated heuristic, not libmagic. It will miss
  and occasionally mislabel formats.
- The block classifier and file verdict are guesses derived from entropy and
  byte statistics. Treat them as a lead, not a conclusion.
- UTF-16 string extraction is little-endian only.
- `.deb` packages are unsigned pending release-signing infrastructure.
