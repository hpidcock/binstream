package main

func ubuntuArch(arch string) string {
	switch arch {
	case "ppc64", "ppc64le":
		return "ppc64el"
	default:
		return arch
	}
}
