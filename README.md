Secure SHell for Alfred
=======================

Open SSH connections from [Alfred 3][alfredapp] with autosuggestions based on `~/.ssh/known_hosts`, `/etc/hosts` and your history.

!["Secure SHell Demo"][demo]


Features
--------

- Auto-suggest hostnames from `/etc/hosts` and `~/.ssh/known_hosts` (sources can be individually disabled).
- Remembers usernames, so you don't have to type them in every time. (You can also remove connections from your history or disable it entirely.)
- Alternate actions:
  - Open connection with mosh instead of SSH.
  - Open SFTP connection instead of SSH.
  - Ping host.


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


**Please note**: The workflow generates an `ssh://` (or `sftp://`) URL and asks Alfred to open it. Similarly, the `ping` and `mosh` features uses Alfred 3's Terminal Command feature. If it's not opening in the right app, it's not the workflow's fault.


Licencing & thanks
------------------

This workflow is released under the [MIT Licence][mit].

It uses the following libraries and resources:

- The icon is based on [Octicons][octicons] by [Github][gh] ([SIL Licence][sil]).
- [ssh_config][ssh_config] ([MIT Licence][mit]) to parse SSH config files.
- [awgo][awgo] ([MIT Licence][mit]) for the workflowy stuff.

This workflow started as a port of [@isometry's][isometry] Python [SSH workflow][ssh-breathe] to Go as a testbed for [awgo][awgo]. It has since gained some additional features.

If you need Alfred 2 support, check out [@isometry's workflow][ssh-breathe].


Changelog
---------

- v.0.5.0 — 2016-10-31
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
[awgo]: https://godoc.org/gogs.deanishe.net/deanishe/awgo
[demo]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/demo.gif
[gh-releases]: https://github.com/deanishe/alfred-ssh/releases/latest
[gh]: https://github.com/
[isometry]: https://github.com/isometry
[mit]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/LICENCE.txt
[octicons]: https://octicons.github.com/
[sil]: http://scripts.sil.org/OFL
[ssh_config]: https://github.com/havoc-io/ssh_config
[ssh-breathe]: https://github.com/isometry/alfredworkflows/tree/master/net.isometry.alfred.ssh
