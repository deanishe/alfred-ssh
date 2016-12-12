Secure SHell for Alfred
=======================

Open SSH/SFTP/mosh connections from [Alfred 3][alfredapp] with autosuggestions based on SSH config files, `/etc/hosts` and your history.

!["Secure SHell Demo"][demo]


Features
--------

- Auto-suggest hostnames from `~/.ssh/*` and `/etc/hosts` (sources can be individually disabled).
- Remembers usernames, so you don't have to type them in every time. (You can also remove connections from your history or disable it entirely.)
- Alternate actions:
  - Open connection with mosh instead of SSH.
  - Open SFTP connection instead of SSH.
  - Ping host.


### Data sources

The workflow reads hosts from the following sources (in this order of priority):

1. `~/.ssh/config`
2. History (i.e. username + host addresses previously entered by the user)
3. `~/.ssh/known_hosts`
4. `/etc/hosts`
5. `/etc/ssh/ssh_config`


Installation
------------

Download [the latest release][gh-releases] and double-click the file to install in Alfred.


Usage
-----

Keyword is `ssh`:

- `ssh [<query>]` — View and filter known SSH connections.

  - `↩` or `⌘+<NUM>` — Open the connection.
  - `⇥` — Expand query to selected connection's title. Useful for adding a port number.
  - `⌘+↩` — Open an SFTP connection instead.
  - `⌘+⌥` — Open a mosh connection instead.
  - `⇧+↩` — Ping host.
  - `^+↩` — Forget connection (if it's from history).


### Configuration

There are several options available in the workflow's configuration sheet. Notably, you can turn off individual autosuggestion sources.

| Variable              | Description                                                                                                                                       |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------|
| `DISABLE_CONFIG`      | Set to `1` to turn off suggestions from `~/.ssh/config`.                                                                                          |
| `DISABLE_ETC_CONFIG`  | Set to `1` to turn off suggestions from `/etc/ssh/ssh_config`.                                                                                    |
| `DISABLE_ETC_HOSTS`   | Set to `1` to turn off suggestions from `/etc/hosts`.                                                                                             |
| `DISABLE_HISTORY`     | Set to `1` to disable the History (reading and writing).                                                                                          |
| `DISABLE_KNOWN_HOSTS` | Set to `1` to turn off suggestions from `~/.ssh/known_hosts`.                                                                                     |
| `EXIT_ON_SUCCESS`     | Set to `1` (default) to close shell if `ping` or `mosh` command exits cleanly                                                                     |
| `EXTERNAL_TRIGGER`    | Set to `1` to use an External Trigger instead of AppleScript to re-open Alfred. The External Trigger is safer, but leaves Alfred in a weird mode. |
| `MOSH_CMD`            | Set to the full path to `mosh` if your shell can't find it. Set to empty to disable `mosh` connections.                                           |

<!--
**Please note**: The workflow generates an `ssh://` (or `sftp://`) URL and asks Alfred to open it. Similarly, the `ping` and `mosh` features uses Alfred 3's Terminal Command feature. If it's not opening in the right app, it's not the workflow's fault.
-->

For SSH and SFTP connections, the workflow creates an `ssh://` (or `sftp://`) URL and asks the system to open it. These will open in whichever application you have configured to handle these URLs (Terminal.app is the default for `ssh://`).

The `ping` and `mosh` commands use Alfred's [Terminal Command][alfterm] output, which also call Terminal.app by default.


#### Using iTerm2

If you'd prefer to use iTerm2 rather than Terminal.app, there are two steps:

1. To have `ping` and `mosh` commands open in iTerm2, install [@stuartcryan][stuart]'s [iTerm2 plugin for Alfred][iTerm2-plugin].
2. To open `ssh:` connections in iTerm2, Set iTerm2 as the default handler for `ssh:` URLs in iTerm2's own preferences under `Profiles > PROFILE_NAME > General > URL Schemes`:

![iTerm2 > Preferences > PROFILE_NAME > General > URL Schemes][iTerm2-screenshot]


Licencing & thanks
------------------

This workflow is released under the [MIT Licence][mit].

It uses the following libraries and resources:

- [ssh_config][ssh_config] ([MIT Licence][mit]) by [havoc-io][havoc-io] to parse SSH config files.
- [awgo][awgo] ([MIT Licence][mit]) for the workflowy stuff.
- The icon is based on [Octicons][octicons] ([SIL Licence][sil]) by [Github][gh].

This workflow started as a port of [@isometry's][isometry] Python [SSH workflow][ssh-breathe] to Go as a testbed for [awgo][awgo]. It has since gained some additional features.

If you need Alfred 2 support, check out [@isometry's workflow][ssh-breathe].


Changelog
---------

- v0.7.0 — 2016-12-12
  - Smarter SSH URLs for hosts from `~/.ssh/config`
  - Better removal of duplicates
- v0.6.0 — 2016-11-09
  - Add in-workflow updates
- v0.5.0 — 2016-10-31
  - Add support for SSH configuration files (`~/.ssh/config` and `/etc/ssh/ssh_config`)
  - Alternate action: open connection with `mosh`
- v0.4.0 — 2016-05-27
  - Add ability to turn sources of suggestions off #1
- v0.3.0 — 2016-05-26
  - Alternate action: Open SFTP connection
  - Alternate action: Ping host
  - Remember connections with usernames, so you don't have to type the username each time
- v0.2.0 — 2016-05-23
  - First public release


[alfredapp]: https://www.alfredapp.com/
[alfterm]: https://www.alfredapp.com/help/features/terminal/
[awgo]: https://godoc.org/gogs.deanishe.net/deanishe/awgo
[demo]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/demo.gif
[gh-releases]: https://github.com/deanishe/alfred-ssh/releases/latest
[gh]: https://github.com/
[havoc-io]: https://github.com/havoc-io
[isometry]: https://github.com/isometry
[iTerm2-plugin]: https://github.com/stuartcryan/custom-iterm-applescripts-for-alfred/
[iTerm2-screenshot]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/iTerm2.png "iTerm2 Preferences"
[mit]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/LICENCE.txt
[octicons]: https://octicons.github.com/
[sil]: http://scripts.sil.org/OFL
[ssh_config]: https://github.com/havoc-io/ssh_config
[ssh-breathe]: https://github.com/isometry/alfredworkflows/tree/master/net.isometry.alfred.ssh
[stuart]: https://github.com/stuartcryan/
