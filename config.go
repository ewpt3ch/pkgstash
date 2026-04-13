package main

type Config struct {
	RepoPath  string
	MirrorURL string
}

func NewConfig() *Config {
	return &Config{
		RepoPath:  "/home/ewpt3ch/dev/pacman-cache-server/tmprepo",
		MirrorURL: "https://us.mirrors.cicku.me/archlinux/",
	}
}
