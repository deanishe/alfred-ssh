Secure SHell for Alfred
=======================

Open SSH connections from [Alfred 3][alfredapp] with autosuggestions based on known hosts and /etc/hosts.

!["Secure SHell Demo"][demo]

This is a port of @isometry's Python [SSH workflow][ssh-breathe] to Alfred 3 and Go.

Usage
-----

Keyword is `ssh`:

- `ssh [<query>]` — View and filter known SSH connections.
    - `↩` or `⌘+<NUM>` — Open the connection.
    - `⇥` — Expand query to selected connection's title. Useful for adding a port number.


Licence
-------

This workflow is released under the [MIT License][mit].

The icon is from [Octicons][octicons] by [Github][gh], released under the [SIL License][sil].


[alfredapp]: https://www.alfredapp.com/
[demo]: https://raw.githubusercontent.com/deanishe/alfred-ssh/master/demo.gif
[octicons]: https://octicons.github.com/
[gh]: https://github.com/
[ssh-breathe]: https://github.com/isometry/alfredworkflows/tree/master/net.isometry.alfred.ssh
[mit]: ./LICENCE.txt
[sil]: http://scripts.sil.org/OFL
