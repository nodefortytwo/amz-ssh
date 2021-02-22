package update

import (
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

//Handler replaces the current running binary with the latest version from github
func Handler(c *cli.Context) error {
	v := semver.MustParse(c.App.Version)
	latest, err := selfupdate.UpdateSelf(v, "nodefortytwo/amz-ssh")
	if err != nil {
		log.Println("Binary update failed:", err)
		return nil
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", c.App.Version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		log.Println("Release note:\n", latest.ReleaseNotes)
	}

	return nil
}
