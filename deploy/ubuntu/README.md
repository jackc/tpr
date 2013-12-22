# Ubuntu

Create tpr user

    adduser --system --quiet --home / --no-create-home --gecos "The Pithy Reader" tpr

Copy public to /usr/share/tpr

Copy tpr binary to /usr/bin

Create /etc/tpr

Place config.yml in /etc/tpr

tpr.conf is an upstart script. Place it in /etc/init

Start the server

    start tpr

Consider using apache to serve static assets and handle SSL. This also lets us listen on port 80 or 443 without any privileges.
