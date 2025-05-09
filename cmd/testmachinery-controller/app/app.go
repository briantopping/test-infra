// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/admission/webhooks"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/health"
	"github.com/gardener/test-infra/pkg/version"
)

func NewTestMachineryControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "testmachinery-controller",
		Short: "TestMachinery controller manages the orchestration of test in multiple testruns",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) {
	o.log.Info(fmt.Sprintf("start Test Machinery with version %s", version.Get().String()))
	fmt.Println(testmachinery.GetConfig().String())
	if testmachinery.IsRunInsecure() {
		o.log.Info("testmachinery is running in insecure mode")
	}

	o.log.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), o.GetManagerOptions())
	if err != nil {
		o.log.Error(err, "unable to setup manager")
		os.Exit(1)
	}
	mgr.GetClient()

	if err := controller.RegisterTestMachineryController(mgr, ctrl.Log, o.configwatcher.GetConfiguration()); err != nil {
		o.log.Error(err, "unable to create controller", "controllers", "Testrun")
		os.Exit(1)
	}

	if len(o.configwatcher.GetConfiguration().Controller.HealthAddr) != 0 {
		if err := mgr.AddHealthzCheck("default", health.Healthz()); err != nil {
			o.log.Error(err, "unable to register default health check")
			os.Exit(1)
		}
	}

	config := o.configwatcher.GetConfiguration()
	if !config.TestMachinery.Local {
		webhooks.StartHealthCheck(ctx, mgr.GetAPIReader(), config.Controller.DependencyHealthCheck.Namespace, config.Controller.DependencyHealthCheck.DeploymentName, config.Controller.DependencyHealthCheck.Interval)
		o.log.Info("Setup webhooks")
		// TODO use https://github.com/kubernetes-sigs/controller-runtime/pull/2998 when it becomes available in the controller-runtime
		if err := builder.WebhookManagedBy(mgr).
			For(&v1beta1.Testrun{}).
			WithValidator(&webhooks.TestRunCustomValidator{Log: logger.Log.WithName("validator")}).
			Complete(); err != nil {
			o.log.Error(err, "unable to create webhook to validate TestRuns")
			os.Exit(1)
		}
	}

	o.log.Info("starting the controller", "controllers", "Testrun")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
}
