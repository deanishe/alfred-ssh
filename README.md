Secure SHell for Alfred
=======================

Open SSH connections from [Alfred 3][alfredapp] (only) with autosuggestions based on `~/.ssh/known_hosts`, `/etc/hosts` and your history.

!["Secure SHell Demo"][demo]
<!-- !["Secure SHell Demo"](./demo.gif) -->


Features
--------

- Auto-suggest hostnames from `/etc/hosts` and `~/.ssh/known_hosts` (sources can be individually disabled).
- Remembers usernames, so you don't have to type them in every time. (You can also remove connections from your history or disable it entirely.)
- Alternate actions:
  - Open SFTP connection instead of SSH.
  - Ping host.

This started as a straight port of [@isometry's][isometry] Python [SSH workflow][ssh-breathe] to Go as a testbed for the language and a Go workflow library. It has since been ported to Alfred 3 only, and gained some additional features.


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
  - `⇧+↩` — Ping host.
  - `⌥+↩` — Forget connection (if it's from history).

### Configuration

There are several options available in the workflow's configuration sheet. Notably, you can turn off individual autosuggestion sources.

| Variable              | Description                                                                                                                                                  |
|:----------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `DISABLE_ETC_HOSTS`   | Set to `1` to turn off suggestions from `/etc/hosts`.                                                                                                        |
| `DISABLE_KNOWN_HOSTS` | Set to `1` to turn off suggestions from `~/.ssh/known_hosts`.                                                                                                |
| `DISABLE_HISTORY`     | Set to `1` to disable the History (reading and writing).                                                                                                     |
| `EXTERNAL_TRIGGER`    | Set to `1` to re-open Alfred via an External Trigger instead of a direct AppleScript call. The External Trigger is safer, but leaves Alfred in a weird mode. |




**Please note**: The workflow simply generates an `ssh://` (or `sftp://`) URL and asks Alfred to open it. Similarly, the ping function uses Alfred 3's Terminal Command feature. If it's not opening in the right app, it's not the workflow's fault.



Licence
-------

This workflow is released under the [MIT License][mit].

The icon is based on [Octicons][octicons] by [Github][gh], released under the [SIL License][sil].


Changelog
---------

- v0.4.0 — 2016-05-27
  - Add ability to turn sources of suggestions off #1

- v0.3.0 — 2016-05-26
  - Alternate action: Open SFTP connection
  - Alternate action: Ping host
  - Remember connections with usernames, so you don't have to type the username each time

- v0.2.0 — 2016-05-23
  - First public release


[alfredapp]: https://www.alfredapp.com/
[demo]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/demo.gif
[octicons]: https://octicons.github.com/
[gh]: https://github.com/
[gh-releases]: https://github.com/deanishe/alfred-ssh/releases/latest
[isometry]: https://github.com/isometry
[ssh-breathe]: https://github.com/isometry/alfredworkflows/tree/master/net.isometry.alfred.ssh
[mit]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/LICENCE.txt
[sil]: http://scripts.sil.org/OFL
