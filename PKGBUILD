# Maintainer: Eric Phillips <eric@ewpt3ch.dev>

pkgname=pkgstash
pkgver=0.0.3
pkgrel=1
pkgdesc='Sparse caching pacman server'
arch=(x86_64)
url='https://github.com/ewpt3ch/pkgstash'
backup=(etc/pkgstash/pkgstash.toml etc/pkgstash/pkgstash.env)
license=(GPL-2.0-only)
provides=(pkgstash)
conflicts=(pkgstash)
source=("https://github.com/ewpt3ch/${pkgname}/releases/download/v${pkgver}/pkgstash-v${pkgver}-$CARCH.tar.gz")
sha256sums=('209a1036c9123c01fc4639ad17e88a6da7ee5d1c3951192e705f17ba590fbeb6')

package() {
  cd deploy

  install -D -m755 pkgstash "$pkgdir"/usr/bin/pkgstash
  install -D -m644 pkgstash.toml.example "$pkgdir"/etc/"$pkgname"/pkgstash.toml
  install -D -m600 pkgstash.env.example "$pkgdir"/etc/"$pkgname"/pkgstash.env
  install -D -m644 pkgstash.service "$pkgdir"/usr/lib/systemd/system/pkgstash.service
  install -D -m644 pkgstash-refresh.service "$pkgdir"/usr/lib/systemd/system/pkgstash-refresh.service
  install -D -m644 pkgstash-refresh.timer "$pkgdir"/usr/lib/systemd/system/pkgstash-refresh.timer
  install -D -m644 pkgstash.sysusers.d "$pkgdir"/usr/lib/sysusers.d/pkgstash.conf
  install -D -m644 pkgstash.tmpfiles.d "$pkgdir"/usr/lib/tmpfiles.d/pkgstash.conf
}
