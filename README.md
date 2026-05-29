# pkgstash

A sparse caching pacman mirror

## Motivation

Reduces external traffic in a local network with multiple arch linux systems with minimal config. It replicates the structure of a full mirror without downloading the whole mirror. Clients add this in mirrorlist and pacman just works. 

## Quick Start

Currently disributed as a binary. Download the PKGBUILD and pkgstash.install into and empty directory, use makepkg -i to install. See [Arch User Repository](https://wiki.archlinux.org/title/Arch_User_Repository) for more information.

### Configure and Start

Edit `/etc/pkgstash/pkgstash.env` and change the token to a secure one. I used openssl like so:

    openssl rand -base64 64

```
#/etc/pkgstash/pkgstash.env

PKGSTASH_TOKEN=<paste secure token here>
```

start with the included systemd service

    systemctl enable --now pkgstash.service

    systemctl enable --now pkgstash-refresh.timer

## Usage

Set your pacman mirrorlist to point at your server and use pacman like Normal. Update mirror urls, mirrored repos in /etc/pkgstash/pkgstash.toml to suit your needs.

### Technical details

1. Go as the language. It has most of the tools necessary for a http server ins the standard library and it's performant
2. Initial usage streams downloads to multiple clients concurrently from one upstream request
3. Pkgs are written atomically to prevent serving partial files
4. Use multiple mirrors for redundancy and spread the load out
5. Sync the cache on a schedule, includes refreshing databases, prefetching cached pkgs, and cleaning out old pkgs

### Roadmap

- cli admin tool
- Custom repo
- Build server
- Automate builds on updated aur pkgs
    - Auto approve on version bump with not other changes in PKGSBUILD
    - Notify if a build needs approval
- Web admin interface

## Contributing

clone the repo from repo root

    go build ./cmd/server

Binary is called pkgstash, sample config, env and service files are in deploy/

