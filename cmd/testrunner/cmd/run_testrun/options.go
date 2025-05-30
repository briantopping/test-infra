// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package run_testrun

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testrunner"
)

type options struct {
	testrunnerConfig testrunner.Config
	watchOptions     watch.Options

	fs                   *pflag.FlagSet
	testrunNamePrefix    string
	tmKubeconfigPath     string
	testrunPath          string
	testrunFlakeAttempts int

	timeout time.Duration
}

// NewOptions creates a new options struct.
func NewOptions() *options {
	d := time.Minute
	return &options{
		testrunnerConfig: testrunner.Config{},
		watchOptions: watch.Options{
			PollInterval: &d,
		},
	}
}

// Validate validates the options
func (o *options) Validate() error {
	if len(o.tmKubeconfigPath) == 0 {
		return errors.New("tm-kubeconfig-path is required")
	}
	if len(o.testrunNamePrefix) == 0 {
		return errors.New("testrun-prefix is required")
	}
	if len(o.testrunPath) == 0 {
		return errors.New("file is required")
	}
	return nil
}

func (o *options) AddFlags(fs *pflag.FlagSet) error {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.StringVar(&o.tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	fs.StringVar(&o.testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	fs.StringVarP(&o.testrunnerConfig.Namespace, "namespace", "n", "default", "Namesapce where the testrun should be deployed.")
	fs.DurationVar(&o.timeout, "timeout", 1*time.Hour, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	fs.IntVar(&o.testrunFlakeAttempts, "testrun-flake-attempts", 0, "Max number of testruns until testrun is successful")
	fs.BoolVar(&o.testrunnerConfig.NoExecutionGroup, "no-execution-group", false, "do not inject a execution group id into testruns")

	fs.StringVarP(&o.testrunPath, "file", "f", "", "Path to the testrun yaml")
	fs.BoolVar(&o.testrunnerConfig.Serial, "serial", false, "executes all testruns of a bucket only after the previous bucket has finished")
	fs.IntVar(&o.testrunnerConfig.BackoffBucket, "backoff-bucket", 0, "Number of parallel created testruns per backoff period")
	fs.DurationVar(&o.testrunnerConfig.BackoffPeriod, "backoff-period", 0, "Time to wait between the creation of testrun buckets")
	fs.DurationVar(o.watchOptions.PollInterval, "poll-interval", time.Minute, "poll interval of the underlaying watch")

	// DEPRECATED FLAGS
	// is now handled by the testmachinery
	fs.Int64("interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")
	fs.String("output-dir-path", "./testout", "The filepath where the summary should be written to.")
	fs.String("es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	fs.String("es-endpoint", "", "endpoint of the elasticsearch instance")
	fs.String("es-username", "", "username to authenticate against a elasticsearch instance")
	fs.String("es-password", "", "password to authenticate against a elasticsearch instance")
	fs.String("s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	fs.Bool("s3-ssl", false, "S3 has SSL enabled.")
	if err := fs.MarkDeprecated("interval", "no interval "); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("output-dir-path", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-config-name", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-endpoint", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-username", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-password", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("s3-endpoint", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("s3-ssl", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}

	o.fs = fs

	return nil
}
