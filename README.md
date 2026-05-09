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

Set your pacman mirrorlist to point at your server and use pacman like Normal. Currently cache misses are not streamed to the client until the server has the whole file, workaround by turning off pacmans timeout.


### Technical details

1. Go as the language. It has most of the tools necessary for a http server ins the standard library and it's performant.
2. Singleflight to prevent 2 clients from causing the server to fetch the same package more than once.
3. Pkgs are written atomically to prevent serving partial files
4. Use multiple mirrors for redundancy and spread the load out.
5. Refresh the remote db files on a schedule to enable prefetching of cached pkgs.

### Roadmap

- Prefetch and cache cleanup
- Streaming on cache misses
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

