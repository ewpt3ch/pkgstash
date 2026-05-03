- Add better logging for errors, filename more deatail
- implement streaming

- Complete testing
- Deployment(PKGBUILD, bootstrap script?)
- More complete sync(refresh packages on schedule with db, prefetch updates to pkgs we already have)
- clean cache of old files
- Add chi for mux
- Build server/tool
- ~retry on failed fetch~
- ~Solve timeout issue large pkgs~
- ~Move project to github as primary~
- ~add a UA to get client, some servers are 403 the default go client~
- ~Basic config Testing~
- ~flag for loading non default config~
- ~add cli option for config location~
- ~Deployment(systemd, systemd.timer)~
- ~Basic testing for internal/cache~
- ~basic file server that fulfills pacman api~
- ~fetch requested files from mirror~
- ~DB sync from mirror~
- ~Config file and import~

Looking at what you've built and what's left on your TODO, here's how I'd think about the "done" line for a capstone:

**What you already have that's impressive:**
- Real concurrent systems work — singleflight, atomic ops, mutex
- Custom HTTP transport with timeout tuning
- Atomic file writes with the tmp-then-rename pattern
- Meaningful test suite with httptest, not just trivial unit tests
- Clean dependency injection, thoughtful package boundaries
- Config validation with proper error wrapping

**What would make it a complete capstone story:**
- The streaming solution for large packages — this is your most interesting unsolved technical problem and directly demonstrates you understand the HTTP layer
- At least one more meaningful feature working end-to-end, either the refresh scheduling via systemd timer or the prefetch-on-refresh logic
- Actually deployed and serving your home network — "running in production" is a strong closer

**What I'd leave out of the capstone scope:**
- The AUR builder — that's a whole second project honestly
- Chi middleware — nice but not substantive enough to matter

The through-line you want to be able to tell is: *I built a caching reverse proxy from scratch in Go, handling concurrency, filesystem atomicity, and HTTP semantics correctly, and deployed it on real hardware.* You're close to being able to say that. The streaming piece and a working deployment are the remaining gaps.

How far out is the capstone deadline?
