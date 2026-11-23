package filedist

import (
	"os"
	"strings"
	"testing"
)

// TestUploadContextRemoteDirUsesPathPinsDocuments the Windows path fix in
// sftp_client.go's UploadContext: the MkdirAll argument for a remote (POSIX)
// path must be derived with path.Dir. On a Windows controller,
// filepath.Dir("/data/pkg.tgz") yields `\data`, while pkg/sftp MkdirAll only
// splits on "/" — so it would create one literal backslash-named directory and
// the dispatch would fail masked as transferFailed. Local downloads (line
// localDir := filepath.Dir(localPath)) must stay on the host separator.
// There is no in-process SFTP server in this package, so the test pins the
// call sites rather than hitting a real endpoint; a revert of either line
// fails this test regardless of the host OS.
func TestUploadContextRemoteDirUsesPathPins(t *testing.T) {
	src, err := os.ReadFile("sftp_client.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	const remote = "remoteDir := path.Dir(remotePath)"
	const local = "localDir := filepath.Dir(localPath)"
	if !strings.Contains(body, remote) {
		t.Fatalf("sftp_client.go lost %q; switching it to filepath.Dir would recreate the Windows garbage-directory bug", remote)
	}
	if strings.Contains(body, "localDir := path.Dir") {
		t.Fatalf("sftp_client.go must keep %q for the local download directory (host separator), not path.Dir", local)
	}
}