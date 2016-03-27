
Secure SHell for Alfred
=======================

Open SSH connections from [Alfred 2][alfredapp] with autosuggestions based on known hosts and /etc/hosts.

This is a reimplementation of @isometry's Python [SSH workflow][ssh-breathe] in Go.

I wanted to see how well Go works as a workflow language compared to Python, and an SSH workflow was a good test project, as it only uses local data.

The bottom line is, Go runs about 20x faster than Python, but as the Python workflow runs in well under 0.1s, that's not such a big deal here.

Also, the Go binary is massive compared to the equivalent Python script (4MB vs 10kB).


Usage
-----

Keyword is `ssh`:

- `ssh [<query>]` — View and filter known SSH connections.
    - `↩ ` — Open the actioned connection.


Licence
-------

This workflow is released under the [MIT License][mit].

The icon is from [Octicons][octicons] by [Github][gh], released under the [SIL License][sil].


[alfredapp]: https://www.alfredapp.com/
[octicons]: https://octicons.github.com/
[gh]: https://github.com/
[ssh-breathe]: https://github.com/isometry/alfredworkflows/tree/master/net.isometry.alfred.ssh
[mit]: ./LICENCE.txt
[sil]: http://scripts.sil.org/OFL

