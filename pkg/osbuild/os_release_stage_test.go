package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/image-builder/pkg/osbuild"
)

func TestOSReleaseStageJsonMinimal(t *testing.T) {
	expectedJson := `{
  "type": "org.osbuild.os-release",
  "options": {
    "vars": {}
  }
}`

	opts := &osbuild.OSReleaseStageOptions{}
	stage := osbuild.NewOSReleaseStage(opts)
	require.NotNil(t, stage)

	js, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, expectedJson, string(js))
}

func TestOSReleaseStageJsonFull(t *testing.T) {
	extensionReleaseStrict := false
	expectedJson := `{
  "type": "org.osbuild.os-release",
  "options": {
    "path": "usr/lib/extension-release.d/extension-release.myext",
    "extension-release-strict": false,
    "vars": {
      "NAME": "Fedora Linux",
      "ID": "fedora",
      "ID_LIKE": "rhel centos",
      "VERSION": "41 (Server Edition)",
      "VERSION_ID": "41",
      "PRETTY_NAME": "Fedora Linux 41 (Server Edition)",
      "ANSI_COLOR": "0;38;2;60;110;180",
      "CPE_NAME": "cpe:/o:fedoraproject:fedora:41",
      "HOME_URL": "https://fedoraproject.org/",
      "DOCUMENTATION_URL": "https://docs.fedoraproject.org/",
      "SUPPORT_URL": "https://ask.fedoraproject.org/",
      "BUG_REPORT_URL": "https://bugzilla.redhat.com/",
      "PRIVACY_POLICY_URL": "https://fedoraproject.org/wiki/Legal:PrivacyPolicy",
      "VARIANT": "Server Edition",
      "VARIANT_ID": "server",
      "LOGO": "fedora-logo-icon",
      "SYSEXT_LEVEL": "1.0",
      "SYSEXT_SCOPE": "system",
      "EXTENSION_RELOAD_MANAGER": "1"
    }
  }
}`

	opts := &osbuild.OSReleaseStageOptions{
		Path:                   "usr/lib/extension-release.d/extension-release.myext",
		ExtensionReleaseStrict: &extensionReleaseStrict,
		Vars: osbuild.OSReleaseStageVars{
			Name:                   "Fedora Linux",
			ID:                     "fedora",
			IDLike:                 "rhel centos",
			Version:                "41 (Server Edition)",
			VersionID:              "41",
			PrettyName:             "Fedora Linux 41 (Server Edition)",
			ANSIColor:              "0;38;2;60;110;180",
			CPEName:                "cpe:/o:fedoraproject:fedora:41",
			HomeURL:                "https://fedoraproject.org/",
			DocumentationURL:       "https://docs.fedoraproject.org/",
			SupportURL:             "https://ask.fedoraproject.org/",
			BugReportURL:           "https://bugzilla.redhat.com/",
			PrivacyPolicyURL:       "https://fedoraproject.org/wiki/Legal:PrivacyPolicy",
			Variant:                "Server Edition",
			VariantID:              "server",
			Logo:                   "fedora-logo-icon",
			SysextLevel:            "1.0",
			SysextScope:            "system",
			ExtensionReloadManager: "1",
		},
	}
	stage := osbuild.NewOSReleaseStage(opts)
	require.NotNil(t, stage)

	js, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, expectedJson, string(js))
}
