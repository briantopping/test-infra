// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package pages

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	github2 "github.com/google/go-github/v72/github"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
)

type IconWithTooltip struct {
	Icon    string
	Tooltip string
	Color   string
}

var StepPhaseIcon = func(phase v1alpha1.NodePhase) IconWithTooltip {
	switch phase {
	case v1beta1.StepPhaseInit:
		return RunPhaseIcon(v1beta1.RunPhaseInit)
	case v1beta1.StepPhasePending:
		return RunPhaseIcon(v1beta1.RunPhasePending)
	case v1beta1.StepPhaseRunning:
		return RunPhaseIcon(v1beta1.RunPhaseRunning)
	case v1beta1.StepPhaseSuccess:
		return RunPhaseIcon(v1beta1.RunPhaseSuccess)
	case v1beta1.StepPhaseFailed:
		return RunPhaseIcon(v1beta1.RunPhaseFailed)
	case v1beta1.StepPhaseError:
		return RunPhaseIcon(v1beta1.RunPhaseError)
	case v1beta1.StepPhaseTimeout:
		return RunPhaseIcon(v1beta1.RunPhaseTimeout)
	default:
		return IconWithTooltip{
			Icon:    "info",
			Tooltip: fmt.Sprintf("%s phase", phase),
			Color:   "grey",
		}

	}
}

var RunPhaseIcon = func(phase v1alpha1.WorkflowPhase) IconWithTooltip {
	switch phase {
	case v1beta1.RunPhaseInit:
		return IconWithTooltip{
			Icon:    "schedule",
			Tooltip: fmt.Sprintf("%s phase: Testrun is waiting to be scheduled", v1beta1.StepPhaseInit),
			Color:   "grey",
		}
	case v1beta1.RunPhasePending:
		return IconWithTooltip{
			Icon:    "schedule",
			Tooltip: fmt.Sprintf("%s phase: Testrun is pending", v1beta1.RunPhasePending),
			Color:   "orange",
		}
	case v1beta1.RunPhaseRunning:
		return IconWithTooltip{
			Icon:    "autorenew",
			Tooltip: fmt.Sprintf("%s phase: Testrun is running", v1beta1.RunPhaseRunning),
			Color:   "orange",
		}
	case v1beta1.RunPhaseSuccess:
		return IconWithTooltip{
			Icon:    "done",
			Tooltip: fmt.Sprintf("%s phase: Testrun succeeded", v1beta1.RunPhaseSuccess),
			Color:   "green",
		}
	case v1beta1.RunPhaseFailed:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun failed", v1beta1.RunPhaseFailed),
			Color:   "red",
		}
	case v1beta1.RunPhaseError:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun errored", v1beta1.RunPhaseError),
			Color:   "red",
		}
	case v1beta1.RunPhaseTimeout:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun run longer than the specified timeout", v1beta1.StepPhaseTimeout),
			Color:   "red",
		}
	default:
		return IconWithTooltip{
			Icon:    "info",
			Tooltip: fmt.Sprintf("%s phase", phase),
			Color:   "grey",
		}

	}
}

type runItem struct {
	Organization string
	Repository   string
	PR           int64

	Testrun  string
	Phase    IconWithTooltip
	Progress string

	ArgoURL string
}

type runDetailedItem struct {
	runItem
	Author    string
	StartTime string
	RawStatus string
}

func NewPRStatusPage(p *Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAuthenticated := true
		_, err := p.auth.GetAuthContext(r)
		if err != nil {
			p.log.V(3).Info(err.Error())
			isAuthenticated = false
		}

		allTests := tests.GetAllRunning()
		if len(allTests) == 0 {
			allTests = append(allTests, &demotest)
		}

		rawList := make([]runItem, len(allTests))
		for i, run := range allTests {
			rawList[i] = runItem{
				Organization: run.Event.GetOwnerName(),
				Repository:   run.Event.GetRepositoryName(),
				PR:           run.Event.ID,
				Testrun:      run.Testrun.GetName(),
				Phase:        RunPhaseIcon(util.TestrunStatusPhase(run.Testrun)),
				Progress:     util.TestrunProgress(run.Testrun),
			}
			if isAuthenticated {
				rawList[i].ArgoURL, _ = testrunner.GetArgoURL(context.TODO(), p.runs.GetClient(), run.Testrun)
			}
		}
		params := map[string]interface{}{
			"tests": rawList,
		}

		p.handleSimplePage("pr-status.html", params)(w, r)
	}
}

func NewPRStatusDetailPage(logger logr.Logger, auth auth.Provider, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		trName := mux.Vars(r)["testrun"]
		if trName == "" {
			logger.Info("testrun is not defined")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		allTests := tests.GetAllRunning()
		if len(allTests) == 0 {
			allTests = append(allTests, &demotest)
		}

		var run *tests.Run
		for _, r := range allTests {
			if r.Testrun.GetName() == trName {
				run = r
				break
			}
		}
		if run == nil {
			logger.Error(nil, "testrun cannot be found", "testrun", trName)
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		var (
			statusTable = &strings.Builder{}
			startTime   = ""
		)

		if len(run.Testrun.Status.Steps) != 0 {
			output.RenderStatusTable(statusTable, run.Testrun.Status.Steps)
		}
		if run.Testrun.Status.StartTime != nil {
			startTime = run.Testrun.Status.StartTime.Format(time.RFC822)
		}

		item := runDetailedItem{
			runItem: runItem{
				Organization: run.Event.GetOwnerName(),
				Repository:   run.Event.GetRepositoryName(),
				PR:           run.Event.ID,
				Testrun:      run.Testrun.GetName(),
				Phase:        RunPhaseIcon(util.TestrunStatusPhase(run.Testrun)),
				Progress:     util.TestrunProgress(run.Testrun),
			},
			Author:    run.Event.GetAuthorName(),
			StartTime: startTime,
			RawStatus: statusTable.String(),
		}

		p.handleSimplePage("pr-status-detailed.html", item)(w, r)
	}
}

var owner = "owner"
var repo = "repo"
var author = "demo-user"
var startTime = metav1.NewTime(time.Now())
var demotest = tests.Run{
	Testrun: &v1beta1.Testrun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-tr",
		},
		Status: v1beta1.TestrunStatus{
			StartTime: &startTime,
			Phase:     v1beta1.RunPhaseRunning,
			Steps: []*v1beta1.StepStatus{
				{
					Phase: v1beta1.StepPhaseRunning,
				},
			},
			Workflow: "tm-test49l44",
		},
	},
	Event: &github.GenericRequestEvent{
		ID:     3,
		Number: 0,
		Repository: &github2.Repository{
			Owner: &github2.User{
				Login: &owner,
			},
			Name: &repo,
		},
		Author: &github2.User{Login: &author},
	},
}
