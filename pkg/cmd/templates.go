package cmd

import (
	"embed"
)

var (
	//go:embed templates/*
	TempFS embed.FS
)
