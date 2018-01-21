package steps

import (
	"bytes"
	"encoding/base64"
	"fmt"
)

func waitForContainerCmd(name, shareddir string) string {
	command := `while ! test -f "` + shareddir + `/` + name + `.done"; do
	sleep 1
	done
	`
	return command
}

// generateScript is a helper function that generates a build script and base64 encode it.
func generateScript(commands []string) string {
	var buf bytes.Buffer
	for _, command := range commands {
		buf.WriteString(fmt.Sprintf(`
%s
`, command,
		))
	}
	return base64.StdEncoding.EncodeToString([]byte(buf.String()))
}
