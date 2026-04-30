package mdori

import (
	"fmt"
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	name, args, err := browserCommandFor(runtime.GOOS, url)
	if err != nil {
		return err
	}

	return exec.Command(name, args...).Start()
}

func browserCommandFor(goos, url string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{url}, nil
	case "linux":
		return "xdg-open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q", goos)
	}
}
