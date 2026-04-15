package main

type Config struct {
	MirrorRoot string
	MirrorURL  string
	Port       string
}

func NewConfig() *Config {
	return &Config{
		MirrorRoot: "/home/ewpt3ch/dev/pacman-cache-server/tmprepo",
		MirrorURL:  "https://us.mirrors.cicku.me/archlinux/",
		Port:       "8090",
	}
}
