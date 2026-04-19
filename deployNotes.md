Create /srv/pkgstash/{repo}/os/{arch} for every repo, core and extra at min
Create pkgstash user
chown -R srv/pkgstash
mkdir /etc/pkgstash
cp .toml to /etc/pkgstash
cp binary to /usr/local/bin/
cp systemd files to /usr/local/lib/systemd/system/
