package main

import (
	"fmt"
	"net"
	"time"
)

func (this *Server) Rollback(conn net.Conn, applicationName, version string) error {
	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		deployLock.start()
		defer deployLock.finish()

		if app.LastDeploy == "" {
			return fmt.Errorf("Automatic rollback version detection is impossible because this app has not had any releases")
		}
		if app.LastDeploy == "v1" {
			return fmt.Errorf("Automatic rollback version detection is impossible because this app has only had 1 release")
		}
		if version == "" {
			// Get release before current.
			var err error = nil
			version, err = app.CalcPreviousVersion()
			if err != nil {
				return err
			}
		}
		logger := NewLogger(NewTimeLogger(NewMessageLogger(conn)), "[rollback] ")
		fmt.Fprintf(logger, "Rolling back to %v\n", version)

		// Get the next version.
		app, cfg, err := this.IncrementAppVersion(app)
		if err != nil {
			return err
		}

		deployment := &Deployment{
			Server:      this,
			Logger:      logger,
			Config:      cfg,
			Application: app,
			Version:     app.LastDeploy,
			StartedTs:   time.Now(),
		}

		// Cleanup any hanging chads upon error.
		defer func() {
			if err != nil {
				deployment.undoVersionBump()
			}
		}()

		err = deployment.extract(version)
		if err != nil {
			return err
		}
		err = deployment.archive()
		if err != nil {
			return err
		}
		err = deployment.deploy()
		if err != nil {
			return err
		}
		return nil
	})
}
